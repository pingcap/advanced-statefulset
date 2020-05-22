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

package config

import (
	pcclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/advanced-statefulset/pkg/component/config"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
)

// Config is the main context object for the controller manager.
type Config struct {
	GenericComponent config.GenericComponentConfiguration

	// the general kube client
	Client *clientset.Clientset

	// the general pingcap client
	PCClient *pcclientset.Clientset

	// the client only used for leader election
	// LeaderElectionClient *clientset.Clientset
	// LeaderElection is optional.
	LeaderElection *leaderelection.LeaderElectionConfig

	// the rest config for the master
	Kubeconfig *rest.Config

	// the event sink
	EventRecorder record.EventRecorder
}

type completedConfig struct {
	*Config
}

// CompletedConfig same as Config, just to swap private object.
type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() *CompletedConfig {
	cc := completedConfig{c}
	return &CompletedConfig{&cc}
}
