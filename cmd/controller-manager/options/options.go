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

package options

import (
	"fmt"
	"os"
	"time"

	pcclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	controllermanagerconfig "github.com/pingcap/advanced-statefulset/cmd/controller-manager/config"
	"github.com/pingcap/advanced-statefulset/pkg/component/config"
	"github.com/pingcap/advanced-statefulset/pkg/component/options"
	v1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	cliflag "k8s.io/component-base/cli/flag"
	componentbaseconfig "k8s.io/component-base/config"
	"k8s.io/klog"
)

// ControllerManagerOptions is the main context object for the advanced-statefulset-controller-manager.
type ControllerManagerOptions struct {
	GenericComponent *options.GenericComponentOptions

	Controllers []string

	Master     string
	Kubeconfig string
}

// NewControllerManagerOptions creates a new ControllerManagerOptions with a default config.
func NewControllerManagerOptions() *ControllerManagerOptions {
	genericComponetConfig := config.NewDefaultGenericComponentConfiguration()
	s := ControllerManagerOptions{
		GenericComponent: options.NewGenericComponentOptions(genericComponetConfig),
	}
	return &s
}

func (s *ControllerManagerOptions) Flags() (nfs cliflag.NamedFlagSets) {
	fs := nfs.FlagSet("misc")
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization and master location information.")

	s.GenericComponent.AddFlags(nfs.FlagSet("generic"))
	return
}

// ApplyTo fills up controller manager config with options.
func (s *ControllerManagerOptions) ApplyTo(c *controllermanagerconfig.Config, userAgent string) error {
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

	// CRD does not support protobuf.
	c.Kubeconfig.ContentConfig.ContentType = "application/json"
	c.PCClient, err = pcclientset.NewForConfig(rest.AddUserAgent(c.Kubeconfig, userAgent))
	if err != nil {
		return err
	}
	leaderElectionClient := clientset.NewForConfigOrDie(rest.AddUserAgent(c.Kubeconfig, "leader-election"))

	c.EventRecorder = createRecorder(c.Client, userAgent)

	// Set up leader election if enabled.
	var leaderElectionConfig *leaderelection.LeaderElectionConfig
	if c.GenericComponent.LeaderElection.LeaderElect {
		leaderElectionConfig, err = makeLeaderElectionConfig(c.GenericComponent.LeaderElection, leaderElectionClient, c.EventRecorder)
		if err != nil {
			return err
		}
	}

	c.LeaderElection = leaderElectionConfig
	return nil
}

// Validate is used to validate the options and config before launching the controller manager
func (s *ControllerManagerOptions) Validate() error {
	var errs []error
	errs = append(errs, s.GenericComponent.Validate()...)
	return utilerrors.NewAggregate(errs)
}

// Config configures configuration.
func (s *ControllerManagerOptions) Config() (*controllermanagerconfig.Config, error) {
	c := &controllermanagerconfig.Config{}
	if err := s.ApplyTo(c, "advanced-statefulset-controller-manager"); err != nil {
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

// makeLeaderElectionConfig builds a leader election configuration. It will
// create a new resource lock associated with the configuration.
func makeLeaderElectionConfig(config componentbaseconfig.LeaderElectionConfiguration, client clientset.Interface, recorder record.EventRecorder) (*leaderelection.LeaderElectionConfig, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %v", err)
	}
	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id := hostname + "_" + string(uuid.NewUUID())

	rl, err := resourcelock.New(config.ResourceLock,
		config.ResourceNamespace,
		config.ResourceName,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		return nil, fmt.Errorf("couldn't create resource lock: %v", err)
	}

	return &leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: config.LeaseDuration.Duration,
		RenewDeadline: config.RenewDeadline.Duration,
		RetryPeriod:   config.RetryPeriod.Duration,
		WatchDog:      leaderelection.NewLeaderHealthzAdaptor(time.Second * 20),
		Name:          "advanced-statefulset-controller-manager",
	}, nil
}
