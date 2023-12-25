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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	componentbaseconfig "k8s.io/component-base/config"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

// GenericComponentConfiguration is generic component configuration.
type GenericComponentConfiguration struct {
	// minResyncPeriod is the resync period in reflectors; will be random between
	// minResyncPeriod and 2*minResyncPeriod.
	MinResyncPeriod metav1.Duration
	// contentType is contentType of requests sent to apiserver.
	ContentType string
	// kubeAPIQPS is the QPS to use while talking with kubernetes apiserver.
	KubeAPIQPS float32
	// kubeAPIBurst is the burst to use while talking with kubernetes apiserver.
	KubeAPIBurst int32
	// How long to wait between starting controller managers
	ControllerStartInterval metav1.Duration
	// leaderElection defines the configuration of leader election client.
	LeaderElection componentbaseconfig.LeaderElectionConfiguration
}

// NewDefaultGenericComponentConfiguration returns default GenericComponentConfiguration.
func NewDefaultGenericComponentConfiguration() GenericComponentConfiguration {
	c := GenericComponentConfiguration{
		MinResyncPeriod:         metav1.Duration{Duration: 12 * time.Hour},
		ContentType:             "application/vnd.kubernetes.protobuf",
		KubeAPIQPS:              20,
		KubeAPIBurst:            30,
		ControllerStartInterval: metav1.Duration{Duration: 0 * time.Second},
	}
	leaderElection := componentbaseconfigv1alpha1.LeaderElectionConfiguration{
		// https://github.com/kubernetes/kubernetes/pull/84084
		// https://github.com/kubernetes/kubernetes/pull/106852
		ResourceLock: resourcelock.LeasesResourceLock,
	}
	componentbaseconfigv1alpha1.RecommendedDefaultLeaderElectionConfiguration(&leaderElection)
	componentbaseconfigv1alpha1.Convert_v1alpha1_LeaderElectionConfiguration_To_config_LeaderElectionConfiguration(&leaderElection, &c.LeaderElection, nil)
	return c
}
