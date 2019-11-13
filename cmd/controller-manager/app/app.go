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

package app

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/pingcap/advanced-statefulset/cmd/controller-manager/config"
	"github.com/pingcap/advanced-statefulset/cmd/controller-manager/options"
	pcinformers "github.com/pingcap/advanced-statefulset/pkg/client/informers/externalversions"
	"github.com/pingcap/advanced-statefulset/pkg/controller/statefulset"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/term"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/leaderelection"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/klog"
	utilflag "k8s.io/kubernetes/pkg/util/flag"
)

// ResyncPeriod returns a function which generates a duration each time it is
// invoked; this is so that multiple controllers don't get into lock-step and all
// hammer the apiserver with list requests simultaneously.
func ResyncPeriod(c *config.CompletedConfig) func() time.Duration {
	return func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(c.GenericComponent.MinResyncPeriod.Nanoseconds()) * factor)
	}
}

// Run runs the controller-manager. This should never exit.
func Run(cc *config.CompletedConfig, stopCh <-chan struct{}) error {
	run := func(ctx context.Context) {
		informerFactory := informers.NewSharedInformerFactory(cc.Client, cc.GenericComponent.MinResyncPeriod.Duration)
		pcInformerFactory := pcinformers.NewSharedInformerFactory(cc.PCClient, cc.GenericComponent.MinResyncPeriod.Duration)
		stsCtrl := statefulset.NewStatefulSetController(
			informerFactory.Core().V1().Pods(),
			pcInformerFactory.Apps().V1alpha1().StatefulSets(),
			informerFactory.Core().V1().PersistentVolumeClaims(),
			informerFactory.Apps().V1().ControllerRevisions(),
			cc.Client,
			cc.PCClient,
		)
		go stsCtrl.Run(runtime.NumCPU(), ctx.Done())
		// Start informers after all event listeners are registered.
		informerFactory.Start(ctx.Done())
		pcInformerFactory.Start(ctx.Done())
		<-ctx.Done()
	}

	ctx, cancel := context.WithCancel(context.TODO()) // TODO once Run() accepts a context, it should be used here
	defer cancel()

	go func() {
		select {
		case <-stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// If leader election is enabled, runCommand via LeaderElector until done and exit.
	if cc.LeaderElection != nil {
		cc.LeaderElection.Callbacks = leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				klog.Fatalf("leaderelection lost")
			},
		}
		leaderElector, err := leaderelection.NewLeaderElector(*cc.LeaderElection)
		if err != nil {
			return fmt.Errorf("couldn't create leader elector: %v", err)
		}

		leaderElector.Run(ctx)

		return fmt.Errorf("lost lease")
	}

	run(ctx)
	return fmt.Errorf("finished without leader elect")
}

func NewControllerManagerCommand() *cobra.Command {
	opts := options.NewControllerManagerOptions()
	cmd := &cobra.Command{
		Use:  "controller-manager",
		Long: `Advanced StatefulSet Controller`,
		Run: func(cmd *cobra.Command, args []string) {
			utilflag.PrintFlags(flag.CommandLine)

			c, err := opts.Config()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			if err := Run(c.Complete(), wait.NeverStop); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	namedFlagSets := opts.Flags()
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), cmd.Name())
	for _, f := range namedFlagSets.FlagSets {
		flag.CommandLine.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})

	return cmd
}
