// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Copyright 2019 The Kubernetes Authors.

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

// Modified from https://github.com/kubernetes/kubernetes/blob/v1.17.0/test/e2e/e2e.go

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	e2eutil "github.com/pingcap/advanced-statefulset/test/e2e/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiutilnet "k8s.io/apimachinery/pkg/util/net"
	runtimeutils "k8s.io/apimachinery/pkg/util/runtime"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/version"
	"k8s.io/klog"
	"k8s.io/kubernetes/test/e2e/framework"
	e2ekubectl "k8s.io/kubernetes/test/e2e/framework/kubectl"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	imageutils "k8s.io/kubernetes/test/utils/image"
	utilnet "k8s.io/utils/net"
)

const (
	asNamespace = "advanced-statefulset"
)

var (
	images = []string{
		imageutils.GetE2EImage(imageutils.Httpd),
		imageutils.GetE2EImage(imageutils.HttpdNew),
		imageutils.GetE2EImage(imageutils.Redis),
		imageutils.GetE2EImage(imageutils.Kitten),
		imageutils.GetE2EImage(imageutils.Nautilus),
	}
)

func setupSuite() {
	// Run only on Ginkgo node 1

	c, err := framework.LoadClientset()
	if err != nil {
		klog.Fatal("Error loading client: ", err)
	}

	// Delete any namespaces except those created by the system. This ensures no
	// lingering resources are left over from a previous test run.
	if framework.TestContext.CleanStart {
		deleted, err := framework.DeleteNamespaces(c, nil, /* deleteFilter */
			[]string{
				metav1.NamespaceSystem,
				metav1.NamespaceDefault,
				metav1.NamespacePublic,
				v1.NamespaceNodeLease,
				"local-path-storage",
			})
		if err != nil {
			framework.Failf("Error deleting orphaned namespaces: %v", err)
		}
		klog.Infof("Waiting for deletion of the following namespaces: %v", deleted)
		if err := framework.WaitForNamespacesDeleted(c, deleted, framework.DefaultNamespaceDeletionTimeout); err != nil {
			framework.Failf("Failed to delete orphaned namespaces %v: %v", deleted, err)
		}
	}

	// In large clusters we may get to this point but still have a bunch
	// of nodes without Routes created. Since this would make a node
	// unschedulable, we need to wait until all of them are schedulable.
	framework.ExpectNoError(framework.WaitForAllNodesSchedulable(c, framework.TestContext.NodeSchedulableTimeout))

	// If NumNodes is not specified then auto-detect how many are scheduleable and not tainted
	if framework.TestContext.CloudConfig.NumNodes == framework.DefaultNumNodes {
		nodes, err := e2enode.GetReadySchedulableNodes(c)
		framework.ExpectNoError(err)
		framework.TestContext.CloudConfig.NumNodes = len(nodes.Items)
	}

	// Ensure all pods are running and ready before starting tests (otherwise,
	// cluster infrastructure pods that are being pulled or started can block
	// test pods from running, and tests that ensure all pods are running and
	// ready will fail).
	podStartupTimeout := framework.TestContext.SystemPodsStartupTimeout
	// TODO: In large clusters, we often observe a non-starting pods due to
	// #41007. To avoid those pods preventing the whole test runs (and just
	// wasting the whole run), we allow for some not-ready pods (with the
	// number equal to the number of allowed not-ready nodes).
	if err := e2epod.WaitForPodsRunningReady(c, metav1.NamespaceSystem, int32(framework.TestContext.MinStartupPods), int32(framework.TestContext.AllowedNotReadyNodes), podStartupTimeout, map[string]string{}); err != nil {
		framework.DumpAllNamespaceInfo(c, metav1.NamespaceSystem)
		e2ekubectl.LogFailedContainers(c, metav1.NamespaceSystem, framework.Logf)
		framework.Failf("Error waiting for all pods to be running and ready: %v", err)
	}

	if err := waitForDaemonSets(c, metav1.NamespaceSystem, int32(framework.TestContext.AllowedNotReadyNodes), framework.TestContext.SystemDaemonsetStartupTimeout); err != nil {
		framework.Logf("WARNING: Waiting for all daemonsets to be ready failed: %v", err)
	}

	// Log the version of the server and this client.
	framework.Logf("e2e test version: %s", version.Get().GitVersion)

	dc := c.DiscoveryClient

	serverVersion, serverErr := dc.ServerVersion()
	if serverErr != nil {
		framework.Logf("Unexpected server error retrieving version: %v", serverErr)
	}
	if serverVersion != nil {
		framework.Logf("kube-apiserver version: %s", serverVersion.GitVersion)
	}
}

// waitForDaemonSets for all daemonsets in the given namespace to be ready
// (defined as all but 'allowedNotReadyNodes' pods associated with that
// daemonset are ready).
func waitForDaemonSets(c clientset.Interface, ns string, allowedNotReadyNodes int32, timeout time.Duration) error {
	start := time.Now()
	framework.Logf("Waiting up to %v for all daemonsets in namespace '%s' to start",
		timeout, ns)

	return wait.PollImmediate(framework.Poll, timeout, func() (bool, error) {
		dsList, err := c.AppsV1().DaemonSets(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			framework.Logf("Error getting daemonsets in namespace: '%s': %v", ns, err)
			if IsRetryableAPIError(err) {
				return false, nil
			}
			return false, err
		}
		var notReadyDaemonSets []string
		for _, ds := range dsList.Items {
			framework.Logf("%d / %d pods ready in namespace '%s' in daemonset '%s' (%d seconds elapsed)", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled, ns, ds.ObjectMeta.Name, int(time.Since(start).Seconds()))
			if ds.Status.DesiredNumberScheduled-ds.Status.NumberReady > allowedNotReadyNodes {
				notReadyDaemonSets = append(notReadyDaemonSets, ds.ObjectMeta.Name)
			}
		}

		if len(notReadyDaemonSets) > 0 {
			framework.Logf("there are not ready daemonsets: %v", notReadyDaemonSets)
			return false, nil
		}

		return true, nil
	})
}

// getDefaultClusterIPFamily obtains the default IP family of the cluster
// using the Cluster IP address of the kubernetes service created in the default namespace
// This unequivocally identifies the default IP family because services are single family
// TODO: dual-stack may support multiple families per service
// but we can detect if a cluster is dual stack because pods have two addresses (one per family)
func getDefaultClusterIPFamily(c clientset.Interface) string {
	// Get the ClusterIP of the kubernetes service created in the default namespace
	svc, err := c.CoreV1().Services(metav1.NamespaceDefault).Get(context.TODO(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		framework.Failf("Failed to get kubernetes service ClusterIP: %v", err)
	}

	if utilnet.IsIPv6String(svc.Spec.ClusterIP) {
		return "ipv6"
	}
	return "ipv4"
}

// setupSuitePerGinkgoNode is the boilerplate that can be used to setup ginkgo test suites, on the SynchronizedBeforeSuite step.
// There are certain operations we only want to run once per overall test invocation on each Ginkgo node
// such as making some global variables accessible to all parallel executions
// Because of the way Ginkgo runs tests in parallel, we must use SynchronizedBeforeSuite
// Ref: https://onsi.github.io/ginkgo/#parallel-specs
func setupSuitePerGinkgoNode() {
	// Obtain the default IP family of the cluster
	// Some e2e test are designed to work on IPv4 only, this global variable
	// allows to adapt those tests to work on both IPv4 and IPv6
	// TODO: dual-stack
	// the dual stack clusters can be ipv4-ipv6 or ipv6-ipv4, order matters,
	// and services use the primary IP family by default
	c, err := framework.LoadClientset()
	if err != nil {
		klog.Fatal("Error loading client: ", err)
	}
	framework.TestContext.IPFamily = getDefaultClusterIPFamily(c)
	framework.Logf("Cluster IP family: %s", framework.TestContext.IPFamily)

	// do not check `kube-root-ca.crt` ConfigMap for Kubernetes
	gte119, err := e2eutil.ServerVersionGTE(utilversion.MustParseSemantic("v1.19.0"), c.Discovery())
	framework.ExpectNoError(err)
	if !gte119 {
		framework.TestContext.VerifyServiceAccount = false
	}
}

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	setupSuite()
	// Load images
	kindPath := filepath.Join(framework.TestContext.RepoRoot, "output/bin/kind")
	for _, image := range images {
		e2elog.Logf("Loading image %s", image)
		if err := exec.Command("docker", "pull", image).Run(); err != nil {
			framework.ExpectNoError(err)
		}
		if err := exec.Command(kindPath, "load", "docker-image", "--name", "advanced-statefulset", image).Run(); err != nil {
			framework.ExpectNoError(err)
		}
	}
	// Get the client
	config, err := framework.LoadConfig()
	framework.ExpectNoError(err)
	c, err := clientset.NewForConfig(config)
	framework.ExpectNoError(err, "failed to create clientset")
	// Install CRDs
	framework.RunKubectlOrDie(asNamespace, "apply", "-f", filepath.Join(framework.TestContext.RepoRoot, "manifests/crd.v1.yaml"))
	framework.RunKubectlOrDie(asNamespace, "wait", "--for=condition=Established", "crds/statefulsets.apps.pingcap.com")
	// Install Controller
	_, err = c.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: asNamespace,
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create namespace")
	framework.RunKubectlOrDie(asNamespace, "apply", "-f", filepath.Join(framework.TestContext.RepoRoot, "manifests/rbac.yaml"))
	framework.RunKubectlOrDie(asNamespace, "apply", "-f", filepath.Join(framework.TestContext.RepoRoot, "manifests/deployment.yaml"))
	framework.RunKubectlOrDie(asNamespace, "wait", "--for=condition=Available", "deploy/advanced-statefulset-controller")
	return nil
}, func(data []byte) {
	// Run on all Ginkgo nodes
	setupSuitePerGinkgoNode()
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	CleanupSuite()
}, func() {
	AfterSuiteActions()
})

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func RunE2ETests(t *testing.T) {
	runtimeutils.ReallyCrash = true
	logs.InitLogs()
	defer logs.FlushLogs()

	gomega.RegisterFailHandler(framework.Fail)

	// Disable skipped tests unless they are explicitly requested.
	if config.GinkgoConfig.FocusString == "" && config.GinkgoConfig.SkipString == "" {
		config.GinkgoConfig.SkipString = `\[Flaky\]|\[Feature:.+\]`
	}

	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	var r []ginkgo.Reporter
	if framework.TestContext.ReportDir != "" {
		// TODO: we should probably only be trying to create this directory once
		// rather than once-per-Ginkgo-node.
		if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
			klog.Errorf("Failed creating report directory: %v", err)
		} else {
			r = append(r, reporters.NewJUnitReporter(path.Join(framework.TestContext.ReportDir, fmt.Sprintf("junit_%v%02d.xml", framework.TestContext.ReportPrefix, config.GinkgoConfig.ParallelNode))))
		}
	}
	klog.Infof("Starting e2e run %q on Ginkgo node %d", framework.RunID, config.GinkgoConfig.ParallelNode)

	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Kubernetes e2e suite", r)
}

func IsRetryableAPIError(err error) bool {
	// These errors may indicate a transient error that we can retry in tests.
	if apierrors.IsInternalError(err) || apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) ||
		apierrors.IsTooManyRequests(err) || apiutilnet.IsProbableEOF(err) || apiutilnet.IsConnectionReset(err) {
		return true
	}
	// If the error sends the Retry-After header, we respect it as an explicit confirmation we should retry.
	if _, shouldRetry := apierrors.SuggestsClientDelay(err); shouldRetry {
		return true
	}
	return false
}
