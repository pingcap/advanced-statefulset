/*
Copyright 2016 The Kubernetes Authors.

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

// this file is copied from k8s.io/kubernetes/test/e2e/framework/test_context.go @v1.23.17

package k8s

import (
	"time"
)

// TestContextType contains test settings and global state. Due to
// historic reasons, it is a mixture of items managed by the test
// framework itself, cloud providers and individual tests.
// The goal is to move anything not required by the framework
// into the code which uses the settings.
//
// The recommendation for those settings is:
//   - They are stored in their own context structure or local
//     variables.
//   - The standard `flag` package is used to register them.
//     The flag name should follow the pattern <part1>.<part2>....<partn>
//     where the prefix is unlikely to conflict with other tests or
//     standard packages and each part is in lower camel case. For
//     example, test/e2e/storage/csi/context.go could define
//     storage.csi.numIterations.
//   - framework/config can be used to simplify the registration of
//     multiple options with a single function call:
//     var storageCSI {
//     NumIterations `default:"1" usage:"number of iterations"`
//     }
//     _ config.AddOptions(&storageCSI, "storage.csi")
//   - The direct use Viper in tests is possible, but discouraged because
//     it only works in test suites which use Viper (which is not
//     required) and the supported options cannot be
//     discovered by a test suite user.
//
// Test suite authors can use framework/viper to make all command line
// parameters also configurable via a configuration file.
type TestContextType struct {
	KubeConfig         string
	KubeContext        string
	KubeAPIContentType string
	KubeVolumeDir      string
	CertDir            string
	Host               string
	BearerToken        string `datapolicy:"token"`
	// TODO: Deprecating this over time... instead just use gobindata_util.go , see #23987.
	RepoRoot                string
	DockershimCheckpointDir string
	// ListImages will list off all images that are used then quit
	ListImages bool

	// ListConformanceTests will list off all conformance tests that are available then quit
	ListConformanceTests bool

	// Provider identifies the infrastructure provider (gce, gke, aws)
	Provider string

	// Tooling is the tooling in use (e.g. kops, gke).  Provider is the cloud provider and might not uniquely identify the tooling.
	Tooling string

	CloudConfig    CloudConfig
	KubectlPath    string
	OutputDir      string
	ReportDir      string
	ReportPrefix   string
	Prefix         string
	MinStartupPods int
	// Timeout for waiting for system pods to be running
	SystemPodsStartupTimeout    time.Duration
	EtcdUpgradeStorage          string
	EtcdUpgradeVersion          string
	GCEUpgradeScript            string
	ContainerRuntime            string
	ContainerRuntimeEndpoint    string
	ContainerRuntimeProcessName string
	ContainerRuntimePidFile     string
	// SystemdServices are comma separated list of systemd services the test framework
	// will dump logs for.
	SystemdServices string
	// DumpSystemdJournal controls whether to dump the full systemd journal.
	DumpSystemdJournal       bool
	ImageServiceEndpoint     string
	MasterOSDistro           string
	NodeOSDistro             string
	NodeOSArch               string
	VerifyServiceAccount     bool
	DeleteNamespace          bool
	DeleteNamespaceOnFailure bool
	AllowedNotReadyNodes     int
	CleanStart               bool
	// If set to 'true' or 'all' framework will start a goroutine monitoring resource usage of system add-ons.
	// It will read the data every 30 seconds from all Nodes and print summary during afterEach. If set to 'master'
	// only master Node will be monitored.
	GatherKubeSystemResourceUsageData string
	GatherLogsSizes                   bool
	GatherMetricsAfterTest            string
	GatherSuiteMetricsAfterTest       bool
	MaxNodesToGather                  int
	// If set to 'true' framework will gather ClusterAutoscaler metrics when gathering them for other components.
	IncludeClusterAutoscalerMetrics bool
	// Currently supported values are 'hr' for human-readable and 'json'. It's a comma separated list.
	OutputPrintType string
	// NodeSchedulableTimeout is the timeout for waiting for all nodes to be schedulable.
	NodeSchedulableTimeout time.Duration
	// SystemDaemonsetStartupTimeout is the timeout for waiting for all system daemonsets to be ready.
	SystemDaemonsetStartupTimeout time.Duration
	// CreateTestingNS is responsible for creating namespace used for executing e2e tests.
	// It accepts namespace base name, which will be prepended with e2e prefix, kube client
	// and labels to be applied to a namespace.
	CreateTestingNS CreateTestingNSFn
	// If set to true test will dump data about the namespace in which test was running.
	DumpLogsOnFailure bool
	// Disables dumping cluster log from master and nodes after all tests.
	DisableLogDump bool
	// Path to the GCS artifacts directory to dump logs from nodes. Logexporter gets enabled if this is non-empty.
	LogexporterGCSPath string
	// featureGates is a map of feature names to bools that enable or disable alpha/experimental features.
	FeatureGates map[string]bool
	// Node e2e specific test context
	NodeTestContextType

	// The DNS Domain of the cluster.
	ClusterDNSDomain string

	// The configuration of NodeKiller.
	NodeKiller NodeKillerConfig

	// The Default IP Family of the cluster ("ipv4" or "ipv6")
	IPFamily string

	// NonblockingTaints is the comma-delimeted string given by the user to specify taints which should not stop the test framework from running tests.
	NonblockingTaints string

	// ProgressReportURL is the URL which progress updates will be posted to as tests complete. If empty, no updates are sent.
	ProgressReportURL string

	// SriovdpConfigMapFile is the path to the ConfigMap to configure the SRIOV device plugin on this host.
	SriovdpConfigMapFile string

	// SpecSummaryOutput is the file to write ginkgo.SpecSummary objects to as tests complete. Useful for debugging and test introspection.
	SpecSummaryOutput string

	// DockerConfigFile is a file that contains credentials which can be used to pull images from certain private registries, needed for a test.
	DockerConfigFile string

	// SnapshotControllerPodName is the name used for identifying the snapshot controller pod.
	SnapshotControllerPodName string

	// SnapshotControllerHTTPPort the port used for communicating with the snapshot controller HTTP endpoint.
	SnapshotControllerHTTPPort int

	// RequireDevices makes mandatory on the environment on which tests are run 1+ devices exposed through device plugins.
	// With this enabled The e2e tests requiring devices for their operation can assume that if devices aren't reported, the test can fail
	RequireDevices bool
}


// NodeKillerConfig describes configuration of NodeKiller -- a utility to
// simulate node failures.
type NodeKillerConfig struct {
	// Enabled determines whether NodeKill should do anything at all.
	// All other options below are ignored if Enabled = false.
	Enabled bool
	// FailureRatio is a percentage of all nodes that could fail simultinously.
	FailureRatio float64
	// Interval is time between node failures.
	Interval time.Duration
	// JitterFactor is factor used to jitter node failures.
	// Node will be killed between [Interval, Interval + (1.0 + JitterFactor)].
	JitterFactor float64
	// SimulatedDowntime is a duration between node is killed and recreated.
	SimulatedDowntime time.Duration
	// NodeKillerStopCh is a channel that is used to notify NodeKiller to stop killing nodes.
	NodeKillerStopCh chan struct{}
}

// NodeTestContextType is part of TestContextType, it is shared by all node e2e test.
type NodeTestContextType struct {
	// NodeE2E indicates whether it is running node e2e.
	NodeE2E bool
	// Name of the node to run tests on.
	NodeName string
	// NodeConformance indicates whether the test is running in node conformance mode.
	NodeConformance bool
	// PrepullImages indicates whether node e2e framework should prepull images.
	PrepullImages bool
	// TODO(pingcap): add this if needed
	// KubeletConfig is the kubelet configuration the test is running against.
	// KubeletConfig kubeletconfig.KubeletConfiguration
	// ImageDescription is the description of the image on which the test is running.
	ImageDescription string
	// RuntimeConfig is a map of API server runtime configuration values.
	RuntimeConfig map[string]string
	// SystemSpecName is the name of the system spec (e.g., gke) that's used in
	// the node e2e test. If empty, the default one (system.DefaultSpec) is
	// used. The system specs are in test/e2e_node/system/specs/.
	SystemSpecName string
	// RestartKubelet restarts Kubelet unit when the process is killed.
	RestartKubelet bool
	// ExtraEnvs is a map of environment names to values.
	ExtraEnvs map[string]string
}

// CloudConfig holds the cloud configuration for e2e test suites.
type CloudConfig struct {
	APIEndpoint       string
	ProjectID         string
	Zone              string   // for multizone tests, arbitrarily chosen zone
	Zones             []string // for multizone tests, use this set of zones instead of querying the cloud provider. Must include Zone.
	Region            string
	MultiZone         bool
	MultiMaster       bool
	Cluster           string
	MasterName        string
	NodeInstanceGroup string // comma-delimited list of groups' names
	NumNodes          int
	ClusterIPRange    string
	ClusterTag        string
	Network           string
	ConfigFile        string // for azure and openstack
	NodeTag           string
	MasterTag         string

	Provider ProviderInterface
}

// TestContext should be used by all tests to access common context data.
var TestContext TestContextType
