package options

import (
	"github.com/cofyc/advanced-statefulset/pkg/component/config"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbasev1alpha1 "k8s.io/component-base/config/v1alpha1"
)

// GenericComponentOptions holds the options which are generic.
type GenericComponentOptions struct {
	MinResyncPeriod         metav1.Duration
	ContentType             string
	KubeAPIQPS              float32
	KubeAPIBurst            int32
	ControllerStartInterval metav1.Duration
	LeaderElection          componentbasev1alpha1.LeaderElectionConfiguration
}

// NewGenericComponentOptions returns generic configuration default
// values.
func NewGenericComponentOptions(cfg config.GenericComponentConfiguration) *GenericComponentOptions {
	o := &GenericComponentOptions{
		MinResyncPeriod:         cfg.MinResyncPeriod,
		ContentType:             cfg.ContentType,
		KubeAPIQPS:              cfg.KubeAPIQPS,
		KubeAPIBurst:            cfg.KubeAPIBurst,
		ControllerStartInterval: cfg.ControllerStartInterval,
		LeaderElection:          cfg.LeaderElection,
	}
	return o
}

// AddFlags adds flags related to generic for controller manager to the specified FlagSet.
func (o *GenericComponentOptions) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.DurationVar(&o.MinResyncPeriod.Duration, "min-resync-period", o.MinResyncPeriod.Duration, "The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod.")
	fs.StringVar(&o.ContentType, "kube-api-content-type", o.ContentType, "Content type of requests sent to apiserver.")
	fs.Float32Var(&o.KubeAPIQPS, "kube-api-qps", o.KubeAPIQPS, "QPS to use while talking with kubernetes apiserver.")
	fs.Int32Var(&o.KubeAPIBurst, "kube-api-burst", o.KubeAPIBurst, "Burst to use while talking with kubernetes apiserver.")
	fs.DurationVar(&o.ControllerStartInterval.Duration, "controller-start-interval", o.ControllerStartInterval.Duration, "Interval between starting controller managers.")

	bindLeaderElectionFlags(&o.LeaderElection, fs)
}

// Validate checks validation of GenericComponentOptions.
func (o *GenericComponentOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errs := []error{}
	return errs
}

// ApplyTo fills up generic config with options.
func (o *GenericComponentOptions) ApplyTo(cfg *config.GenericComponentConfiguration) error {
	if o == nil {
		return nil
	}

	cfg.MinResyncPeriod = o.MinResyncPeriod
	cfg.ContentType = o.ContentType
	cfg.KubeAPIQPS = o.KubeAPIQPS
	cfg.KubeAPIBurst = o.KubeAPIBurst
	cfg.ControllerStartInterval = o.ControllerStartInterval
	cfg.LeaderElection = o.LeaderElection

	return nil
}

// bindLeaderElectionFlags binds the common LeaderElectionCLIConfig flags to a flagset
func bindLeaderElectionFlags(l *componentbasev1alpha1.LeaderElectionConfiguration, fs *pflag.FlagSet) {
	fs.BoolVar(l.LeaderElect, "leader-elect", *l.LeaderElect, ""+
		"Start a leader election client and gain leadership before "+
		"executing the main loop. Enable this when running replicated "+
		"components for high availability.")
	fs.DurationVar(&l.LeaseDuration.Duration, "leader-elect-lease-duration", l.LeaseDuration.Duration, ""+
		"The duration that non-leader candidates will wait after observing a leadership "+
		"renewal until attempting to acquire leadership of a led but unrenewed leader "+
		"slot. This is effectively the maximum duration that a leader can be stopped "+
		"before it is replaced by another candidate. This is only applicable if leader "+
		"election is enabled.")
	fs.DurationVar(&l.RenewDeadline.Duration, "leader-elect-renew-deadline", l.RenewDeadline.Duration, ""+
		"The interval between attempts by the acting master to renew a leadership slot "+
		"before it stops leading. This must be less than or equal to the lease duration. "+
		"This is only applicable if leader election is enabled.")
	fs.DurationVar(&l.RetryPeriod.Duration, "leader-elect-retry-period", l.RetryPeriod.Duration, ""+
		"The duration the clients should wait between attempting acquisition and renewal "+
		"of a leadership. This is only applicable if leader election is enabled.")
	fs.StringVar(&l.ResourceLock, "leader-elect-resource-lock", l.ResourceLock, ""+
		"The type of resource object that is used for locking during "+
		"leader election. Supported options are `endpoints` (default) and `configmaps`.")
}
