/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package framework contains provider-independent helper code for
// building and running E2E tests with Ginkgo. The actual Ginkgo test
// suites gets assembled by combining this framework, the optional
// provider support code and specific tests via a separate .go file
// like Kubernetes' test/e2e.go.

// this file is copied from k8s.io/kubernetes/test/e2e/framework/framework.go @v1.23.17

package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	scaleclient "k8s.io/client-go/scale"
)

const (
	// DefaultNamespaceDeletionTimeout is timeout duration for waiting for a namespace deletion.
	DefaultNamespaceDeletionTimeout = 5 * time.Minute
)

// Framework supports common operations used by e2e tests; it will keep a client & a namespace for you.
// Eventual goal is to merge this with integration test framework.
type Framework struct {
	BaseName string

	// Set together with creating the ClientSet and the namespace.
	// Guaranteed to be unique in the cluster even when running the same
	// test multiple times in parallel.
	UniqueName string

	clientConfig                     *rest.Config
	ClientSet                        clientset.Interface
	KubemarkExternalClusterClientSet clientset.Interface

	DynamicClient dynamic.Interface

	ScalesGetter scaleclient.ScalesGetter

	SkipNamespaceCreation    bool            // Whether to skip creating a namespace
	Namespace                *v1.Namespace   // Every test has at least one namespace unless creation is skipped
	namespacesToDelete       []*v1.Namespace // Some tests have more than one.
	NamespaceDeletionTimeout time.Duration
	SkipPrivilegedPSPBinding bool // Whether to skip creating a binding to the privileged PSP in the test namespace

	gatherer *ContainerResourceGatherer
	// Constraints that passed to a check which is executed after data is gathered to
	// see if 99% of results are within acceptable bounds. It has to be injected in the test,
	// as expectations vary greatly. Constraints are grouped by the container names.
	AddonResourceConstraints map[string]ResourceConstraint

	logsSizeWaitGroup    sync.WaitGroup
	logsSizeCloseChannel chan bool
	logsSizeVerifier     *LogsSizeVerifier

	// Flaky operation failures in an e2e test can be captured through this.
	flakeReport *FlakeReport

	// To make sure that this framework cleans up after itself, no matter what,
	// we install a Cleanup action before each test and clear it after.  If we
	// should abort, the AfterSuite hook should run all Cleanup actions.
	cleanupHandle CleanupActionHandle

	// afterEaches is a map of name to function to be called after each test.  These are not
	// cleared.  The call order is randomized so that no dependencies can grow between
	// the various afterEaches
	afterEaches map[string]AfterEachActionFunc

	// beforeEachStarted indicates that BeforeEach has started
	beforeEachStarted bool

	// configuration for framework's client
	Options Options

	// Place where various additional data is stored during test run to be printed to ReportDir,
	// or stdout if ReportDir is not set once test ends.
	TestSummaries []TestDataSummary

	// TODO(pingcap): add this if needed
	// Place to keep ClusterAutoscaler metrics from before test in order to compute delta.
	// clusterAutoscalerMetricsBeforeTest e2emetrics.Collection

	// Timeouts contains the custom timeouts used during the test execution.
	Timeouts *TimeoutContext
}

// AfterEachActionFunc is a function that can be called after each test
type AfterEachActionFunc func(f *Framework, failed bool)

// TestDataSummary is an interface for managing test data.
type TestDataSummary interface {
	SummaryKind() string
	PrintHumanReadable() string
	PrintJSON() string
}

// Options is a struct for managing test framework options.
type Options struct {
	ClientQPS    float32
	ClientBurst  int
	GroupVersion *schema.GroupVersion
}

// NewDefaultFramework makes a new framework and sets up a BeforeEach/AfterEach for
// you (you can write additional before/after each functions).
func NewDefaultFramework(baseName string) *Framework {
	options := Options{
		ClientQPS:   20,
		ClientBurst: 50,
	}
	return NewFramework(baseName, options, nil)
}

// NewFramework creates a test framework.
func NewFramework(baseName string, options Options, client clientset.Interface) *Framework {
	f := &Framework{
		BaseName:                 baseName,
		AddonResourceConstraints: make(map[string]ResourceConstraint),
		Options:                  options,
		ClientSet:                client,
		Timeouts:                 NewTimeoutContextWithDefaults(),
	}

	f.AddAfterEach("dumpNamespaceInfo", func(f *Framework, failed bool) {
		if !failed {
			return
		}
		if !TestContext.DumpLogsOnFailure {
			return
		}
		if !f.SkipNamespaceCreation {
			for _, ns := range f.namespacesToDelete {
				DumpAllNamespaceInfo(f.ClientSet, ns.Name)
			}
		}
	})

	ginkgo.BeforeEach(f.BeforeEach)
	ginkgo.AfterEach(f.AfterEach)

	return f
}

// BeforeEach gets a client and makes a namespace.
func (f *Framework) BeforeEach() {
	f.beforeEachStarted = true

	// The fact that we need this feels like a bug in ginkgo.
	// https://github.com/onsi/ginkgo/issues/222
	f.cleanupHandle = AddCleanupAction(f.AfterEach)
	if f.ClientSet == nil {
		ginkgo.By("Creating a kubernetes client")
		config, err := LoadConfig()
		ExpectNoError(err)

		config.QPS = f.Options.ClientQPS
		config.Burst = f.Options.ClientBurst
		if f.Options.GroupVersion != nil {
			config.GroupVersion = f.Options.GroupVersion
		}
		if TestContext.KubeAPIContentType != "" {
			config.ContentType = TestContext.KubeAPIContentType
		}
		f.clientConfig = rest.CopyConfig(config)
		f.ClientSet, err = clientset.NewForConfig(config)
		ExpectNoError(err)
		f.DynamicClient, err = dynamic.NewForConfig(config)
		ExpectNoError(err)

		// create scales getter, set GroupVersion and NegotiatedSerializer to default values
		// as they are required when creating a REST client.
		if config.GroupVersion == nil {
			config.GroupVersion = &schema.GroupVersion{}
		}
		if config.NegotiatedSerializer == nil {
			config.NegotiatedSerializer = scheme.Codecs
		}
		restClient, err := rest.RESTClientFor(config)
		ExpectNoError(err)
		discoClient, err := discovery.NewDiscoveryClientForConfig(config)
		ExpectNoError(err)
		cachedDiscoClient := cacheddiscovery.NewMemCacheClient(discoClient)
		restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoClient)
		restMapper.Reset()
		resolver := scaleclient.NewDiscoveryScaleKindResolver(cachedDiscoClient)
		f.ScalesGetter = scaleclient.New(restClient, restMapper, dynamic.LegacyAPIPathResolverFunc, resolver)

		TestContext.CloudConfig.Provider.FrameworkBeforeEach(f)
	}

	if !f.SkipNamespaceCreation {
		ginkgo.By(fmt.Sprintf("Building a namespace api object, basename %s", f.BaseName))
		namespace, err := f.CreateNamespace(f.BaseName, map[string]string{
			"e2e-framework": f.BaseName,
		})
		ExpectNoError(err)

		f.Namespace = namespace

		if TestContext.VerifyServiceAccount {
			ginkgo.By("Waiting for a default service account to be provisioned in namespace")
			err = WaitForDefaultServiceAccountInNamespace(f.ClientSet, namespace.Name)
			ExpectNoError(err)
			ginkgo.By("Waiting for kube-root-ca.crt to be provisioned in namespace")
			err = WaitForKubeRootCAInNamespace(f.ClientSet, namespace.Name)
			ExpectNoError(err)
		} else {
			Logf("Skipping waiting for service account")
		}
		f.UniqueName = f.Namespace.GetName()
	} else {
		// not guaranteed to be unique, but very likely
		f.UniqueName = fmt.Sprintf("%s-%08x", f.BaseName, rand.Int31())
	}

	if TestContext.GatherKubeSystemResourceUsageData != "false" && TestContext.GatherKubeSystemResourceUsageData != "none" {
		var err error
		var nodeMode NodesSet
		switch TestContext.GatherKubeSystemResourceUsageData {
		case "master":
			nodeMode = MasterNodes
		case "masteranddns":
			nodeMode = MasterAndDNSNodes
		default:
			nodeMode = AllNodes
		}

		f.gatherer, err = NewResourceUsageGatherer(f.ClientSet, ResourceGathererOptions{
			InKubemark:                  ProviderIs("kubemark"),
			Nodes:                       nodeMode,
			ResourceDataGatheringPeriod: 60 * time.Second,
			ProbeDuration:               15 * time.Second,
			PrintVerboseLogs:            false,
		}, nil)
		if err != nil {
			Logf("Error while creating NewResourceUsageGatherer: %v", err)
		} else {
			go f.gatherer.StartGatheringData()
		}
	}

	if TestContext.GatherLogsSizes {
		// TODO(pingcap): removed logsSizeVerifier
	}

	// TODO(pingcap): e2emetrics removed, add it back if needed

	f.flakeReport = NewFlakeReport()
}

// printSummaries prints summaries of tests.
func printSummaries(summaries []TestDataSummary, testBaseName string) {
	now := time.Now()
	for i := range summaries {
		Logf("Printing summary: %v", summaries[i].SummaryKind())
		switch TestContext.OutputPrintType {
		case "hr":
			if TestContext.ReportDir == "" {
				Logf(summaries[i].PrintHumanReadable())
			} else {
				// TODO: learn to extract test name and append it to the kind instead of timestamp.
				filePath := path.Join(TestContext.ReportDir, summaries[i].SummaryKind()+"_"+testBaseName+"_"+now.Format(time.RFC3339)+".txt")
				if err := ioutil.WriteFile(filePath, []byte(summaries[i].PrintHumanReadable()), 0644); err != nil {
					Logf("Failed to write file %v with test performance data: %v", filePath, err)
				}
			}
		case "json":
			fallthrough
		default:
			if TestContext.OutputPrintType != "json" {
				Logf("Unknown output type: %v. Printing JSON", TestContext.OutputPrintType)
			}
			if TestContext.ReportDir == "" {
				Logf("%v JSON\n%v", summaries[i].SummaryKind(), summaries[i].PrintJSON())
				Logf("Finished")
			} else {
				// TODO: learn to extract test name and append it to the kind instead of timestamp.
				filePath := path.Join(TestContext.ReportDir, summaries[i].SummaryKind()+"_"+testBaseName+"_"+now.Format(time.RFC3339)+".json")
				Logf("Writing to %s", filePath)
				if err := ioutil.WriteFile(filePath, []byte(summaries[i].PrintJSON()), 0644); err != nil {
					Logf("Failed to write file %v with test performance data: %v", filePath, err)
				}
			}
		}
	}
}

// AddAfterEach is a way to add a function to be called after every test.  The execution order is intentionally random
// to avoid growing dependencies.  If you register the same name twice, it is a coding error and will panic.
func (f *Framework) AddAfterEach(name string, fn AfterEachActionFunc) {
	if _, ok := f.afterEaches[name]; ok {
		panic(fmt.Sprintf("%q is already registered", name))
	}

	if f.afterEaches == nil {
		f.afterEaches = map[string]AfterEachActionFunc{}
	}
	f.afterEaches[name] = fn
}

// AfterEach deletes the namespace, after reading its events.
func (f *Framework) AfterEach() {
	// If BeforeEach never started AfterEach should be skipped.
	// Currently some tests under e2e/storage have this condition.
	if !f.beforeEachStarted {
		return
	}

	RemoveCleanupAction(f.cleanupHandle)

	// This should not happen. Given ClientSet is a public field a test must have updated it!
	// Error out early before any API calls during cleanup.
	if f.ClientSet == nil {
		Failf("The framework ClientSet must not be nil at this point")
	}

	// DeleteNamespace at the very end in defer, to avoid any
	// expectation failures preventing deleting the namespace.
	defer func() {
		nsDeletionErrors := map[string]error{}
		// Whether to delete namespace is determined by 3 factors: delete-namespace flag, delete-namespace-on-failure flag and the test result
		// if delete-namespace set to false, namespace will always be preserved.
		// if delete-namespace is true and delete-namespace-on-failure is false, namespace will be preserved if test failed.
		if TestContext.DeleteNamespace && (TestContext.DeleteNamespaceOnFailure || !ginkgo.CurrentGinkgoTestDescription().Failed) {
			for _, ns := range f.namespacesToDelete {
				ginkgo.By(fmt.Sprintf("Destroying namespace %q for this suite.", ns.Name))
				if err := f.ClientSet.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{}); err != nil {
					if !apierrors.IsNotFound(err) {
						nsDeletionErrors[ns.Name] = err

						// Dump namespace if we are unable to delete the namespace and the dump was not already performed.
						if !ginkgo.CurrentGinkgoTestDescription().Failed && TestContext.DumpLogsOnFailure {
							DumpAllNamespaceInfo(f.ClientSet, ns.Name)
						}
					} else {
						Logf("Namespace %v was already deleted", ns.Name)
					}
				}
			}
		} else {
			if !TestContext.DeleteNamespace {
				Logf("Found DeleteNamespace=false, skipping namespace deletion!")
			} else {
				Logf("Found DeleteNamespaceOnFailure=false and current test failed, skipping namespace deletion!")
			}
		}

		// Paranoia-- prevent reuse!
		f.Namespace = nil
		f.clientConfig = nil
		f.ClientSet = nil
		f.namespacesToDelete = nil

		// if we had errors deleting, report them now.
		if len(nsDeletionErrors) != 0 {
			messages := []string{}
			for namespaceKey, namespaceErr := range nsDeletionErrors {
				messages = append(messages, fmt.Sprintf("Couldn't delete ns: %q: %s (%#v)", namespaceKey, namespaceErr, namespaceErr))
			}
			Failf(strings.Join(messages, ","))
		}
	}()

	// run all aftereach functions in random order to ensure no dependencies grow
	for _, afterEachFn := range f.afterEaches {
		afterEachFn(f, ginkgo.CurrentGinkgoTestDescription().Failed)
	}

	if TestContext.GatherKubeSystemResourceUsageData != "false" && TestContext.GatherKubeSystemResourceUsageData != "none" && f.gatherer != nil {
		// TODO(pingcap): removed gatherer
	}

	if TestContext.GatherLogsSizes {
		// TODO(pingcap): removed logsSizeVerifier
	}

	if TestContext.GatherMetricsAfterTest != "false" {
		// TODO(pingcap): removed metrics grabber
	}

	TestContext.CloudConfig.Provider.FrameworkAfterEach(f)

	// Report any flakes that were observed in the e2e test and reset.
	if f.flakeReport != nil && f.flakeReport.GetFlakeCount() > 0 {
		f.TestSummaries = append(f.TestSummaries, f.flakeReport)
		f.flakeReport = nil
	}

	printSummaries(f.TestSummaries, f.BaseName)

	// Check whether all nodes are ready after the test.
	// This is explicitly done at the very end of the test, to avoid
	// e.g. not removing namespace in case of this failure.
	if err := AllNodesReady(f.ClientSet, 3*time.Minute); err != nil {
		Failf("All nodes should be ready after test, %v", err)
	}
}

// DeleteNamespace can be used to delete a namespace. Additionally it can be used to
// dump namespace information so as it can be used as an alternative of framework
// deleting the namespace towards the end.
func (f *Framework) DeleteNamespace(name string) {
	defer func() {
		err := f.ClientSet.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			Logf("error deleting namespace %s: %v", name, err)
			return
		}
		err = WaitForNamespacesDeleted(f.ClientSet, []string{name}, DefaultNamespaceDeletionTimeout)
		if err != nil {
			Logf("error deleting namespace %s: %v", name, err)
			return
		}
		// remove deleted namespace from namespacesToDelete map
		for i, ns := range f.namespacesToDelete {
			if ns == nil {
				continue
			}
			if ns.Name == name {
				f.namespacesToDelete = append(f.namespacesToDelete[:i], f.namespacesToDelete[i+1:]...)
			}
		}
	}()
	// if current test failed then we should dump namespace information
	if !f.SkipNamespaceCreation && ginkgo.CurrentGinkgoTestDescription().Failed && TestContext.DumpLogsOnFailure {
		DumpAllNamespaceInfo(f.ClientSet, name)
	}

}

// CreateNamespace creates a namespace for e2e testing.
func (f *Framework) CreateNamespace(baseName string, labels map[string]string) (*v1.Namespace, error) {
	createTestingNS := TestContext.CreateTestingNS
	if createTestingNS == nil {
		createTestingNS = CreateTestingNS
	}
	ns, err := createTestingNS(baseName, f.ClientSet, labels)
	// check ns instead of err to see if it's nil as we may
	// fail to create serviceAccount in it.
	f.AddNamespacesToDelete(ns)

	if err == nil && !f.SkipPrivilegedPSPBinding {
		CreatePrivilegedPSPBinding(f.ClientSet, ns.Name)
	}

	return ns, err
}

// AddNamespacesToDelete adds one or more namespaces to be deleted when the test
// completes.
func (f *Framework) AddNamespacesToDelete(namespaces ...*v1.Namespace) {
	for _, ns := range namespaces {
		if ns == nil {
			continue
		}
		f.namespacesToDelete = append(f.namespacesToDelete, ns)

	}
}

// ConformanceIt is wrapper function for ginkgo It.  Adds "[Conformance]" tag and makes static analysis easier.
func ConformanceIt(text string, body interface{}, timeout ...float64) bool {
	return ginkgo.It(text+" [Conformance]", body, timeout...)
}
