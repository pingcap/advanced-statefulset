package options

import (
	controllermanagerconfig "github.com/cofyc/advanced-statefulset/cmd/controller-manager/config"
	pcclientset "github.com/cofyc/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/cofyc/advanced-statefulset/pkg/component/config"
	"github.com/cofyc/advanced-statefulset/pkg/component/options"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

// K8sRebalancerOptions is the main context object for the kirk-controller-manager.
type K8sRebalancerOptions struct {
	GenericComponent *options.GenericComponentOptions

	Controllers []string

	Master     string
	Kubeconfig string
}

// NewK8sRebalancerOptions creates a new K8sRebalancerOptions with a default config.
func NewK8sRebalancerOptions() *K8sRebalancerOptions {
	genericComponetConfig := config.NewDefaultGenericComponentConfiguration()
	s := K8sRebalancerOptions{
		GenericComponent: options.NewGenericComponentOptions(genericComponetConfig),
	}
	return &s
}

// AddFlags adds flags for a specific K8sRebalancerOptions to the specified FlagSet
func (s *K8sRebalancerOptions) AddFlags(fs *pflag.FlagSet) {
	s.GenericComponent.AddFlags(fs)
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization and master location information.")
}

// ApplyTo fills up controller manager config with options.
func (s *K8sRebalancerOptions) ApplyTo(c *controllermanagerconfig.Config, userAgent string) error {
	if err := s.GenericComponent.ApplyTo(&c.GenericComponent); err != nil {
		return err
	}

	var err error
	c.Kubeconfig, err = clientcmd.BuildConfigFromFlags(s.Master, s.Kubeconfig)
	if err != nil {
		return err
	}
	c.Kubeconfig.ContentConfig.ContentType = s.GenericComponent.ContentType
	c.Kubeconfig.QPS = s.GenericComponent.KubeAPIQPS
	c.Kubeconfig.Burst = int(s.GenericComponent.KubeAPIBurst)

	c.Client, err = clientset.NewForConfig(rest.AddUserAgent(c.Kubeconfig, userAgent))
	if err != nil {
		return err
	}

	// FIXME: use protobuf?
	c.Kubeconfig.ContentConfig.ContentType = "application/json"
	c.PCClient, err = pcclientset.NewForConfig(rest.AddUserAgent(c.Kubeconfig, userAgent))
	if err != nil {
		return err
	}

	c.LeaderElectionClient = clientset.NewForConfigOrDie(rest.AddUserAgent(c.Kubeconfig, "leader-election"))

	c.EventRecorder = createRecorder(c.Client, userAgent)
	return nil
}

// Validate is used to validate the options and config before launching the controller manager
func (s *K8sRebalancerOptions) Validate() error {
	var errs []error
	errs = append(errs, s.GenericComponent.Validate()...)
	return utilerrors.NewAggregate(errs)
}

// Config configures configuration.
func (s *K8sRebalancerOptions) Config() (*controllermanagerconfig.Config, error) {
	c := &controllermanagerconfig.Config{}
	if err := s.ApplyTo(c, "advanced-statefulset"); err != nil {
		return nil, err
	}

	return c, nil
}

// createRecorder creates event recorder.
func createRecorder(kubeClient clientset.Interface, userAgent string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	return eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: userAgent})
}
