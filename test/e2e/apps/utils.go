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

package apps

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/test/e2e/framework"
	e2esset "k8s.io/kubernetes/test/e2e/framework/statefulset"
)

var statefulPodRegex = regexp.MustCompile("(.*)-([0-9]+)$")

func getStatefulPodOrdinal(pod *v1.Pod) int {
	ordinal := -1
	subMatches := statefulPodRegex.FindStringSubmatch(pod.Name)
	if len(subMatches) < 3 {
		return ordinal
	}
	if i, err := strconv.ParseInt(subMatches[2], 10, 32); err == nil {
		ordinal = int(i)
	}
	return ordinal
}

// WaitForPartitionedRollingUpdate works like e2esset.WaitForPartitionedRollingUpdate except it takes delete slots into account.
func WaitForPartitionedRollingUpdate(c clientset.Interface, set *appsv1.StatefulSet) (*appsv1.StatefulSet, *v1.PodList) {
	var pods *v1.PodList
	if set.Spec.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType {
		framework.Failf("StatefulSet %s/%s attempt to wait for partitioned update with updateStrategy %s",
			set.Namespace,
			set.Name,
			set.Spec.UpdateStrategy.Type)
	}
	if set.Spec.UpdateStrategy.RollingUpdate == nil || set.Spec.UpdateStrategy.RollingUpdate.Partition == nil {
		framework.Failf("StatefulSet %s/%s attempt to wait for partitioned update with nil RollingUpdate or nil Partition",
			set.Namespace,
			set.Name)
	}
	e2esset.WaitForState(c, set, func(set2 *appsv1.StatefulSet, pods2 *v1.PodList) (bool, error) {
		set = set2
		pods = pods2
		partition := int(*set.Spec.UpdateStrategy.RollingUpdate.Partition)
		if len(pods.Items) < int(*set.Spec.Replicas) {
			return false, nil
		}
		if partition <= 0 && set.Status.UpdateRevision != set.Status.CurrentRevision {
			framework.Logf("Waiting for StatefulSet %s/%s to complete update",
				set.Namespace,
				set.Name,
			)
			e2esset.SortStatefulPods(pods)
			for i := range pods.Items {
				if pods.Items[i].Labels[appsv1.StatefulSetRevisionLabel] != set.Status.UpdateRevision {
					framework.Logf("Waiting for Pod %s/%s to have revision %s update revision %s",
						pods.Items[i].Namespace,
						pods.Items[i].Name,
						set.Status.UpdateRevision,
						pods.Items[i].Labels[appsv1.StatefulSetRevisionLabel])
				}
			}
			return false, nil
		}
		for _, pod := range pods.Items {
			i := getStatefulPodOrdinal(&pod)
			if i < partition {
				// ignored
				continue
			}
			if pod.Labels[appsv1.StatefulSetRevisionLabel] != set.Status.UpdateRevision {
				framework.Logf("Waiting for Pod %s/%s to have revision %s update revision %s",
					pod.Namespace,
					pod.Name,
					set.Status.UpdateRevision,
					pod.Labels[appsv1.StatefulSetRevisionLabel])
				return false, nil
			}
		}
		return true, nil
	})
	return set, pods
}

// WaitForRunning works like e2esset.WaitForRunning except it takes delete slots into account.
func WaitForRunning(c clientset.Interface, numPodsRunning, numPodsReady int32, ss *appsv1.StatefulSet) {
	pollErr := wait.PollImmediate(e2esset.StatefulSetPoll, e2esset.StatefulSetTimeout,
		func() (bool, error) {
			podList := e2esset.GetPodList(c, ss)
			e2esset.SortStatefulPods(podList)
			if int32(len(podList.Items)) < numPodsRunning {
				framework.Logf("Found %d stateful pods, waiting for %d", len(podList.Items), numPodsRunning)
				return false, nil
			}
			if int32(len(podList.Items)) > numPodsRunning {
				return false, fmt.Errorf("too many pods scheduled, expected %d got %d", numPodsRunning, len(podList.Items))
			}
			deleteSlots := helper.GetDeleteSlots(ss)
			replicaCount, deleteSlots := helper.GetMaxReplicaCountAndDeleteSlots(int(*ss.Spec.Replicas), deleteSlots)
			for _, p := range podList.Items {
				if deleteSlots.Has(getStatefulPodOrdinal(&p)) {
					return false, fmt.Errorf("unexpected pod ordinal: %d for stateful set %q", getStatefulPodOrdinal(&p), ss.Name)
				}
				shouldBeReady := getStatefulPodOrdinal(&p) < replicaCount
				isReady := podutil.IsPodReady(&p)
				desiredReadiness := shouldBeReady == isReady
				framework.Logf("Waiting for pod %v to enter %v - Ready=%v, currently %v - Ready=%v", p.Name, v1.PodRunning, shouldBeReady, p.Status.Phase, isReady)
				if p.Status.Phase != v1.PodRunning || !desiredReadiness {
					return false, nil
				}
			}
			return true, nil
		})
	if pollErr != nil {
		framework.Failf("Failed waiting for pods to enter running: %v", pollErr)
	}
}

// WaitForRunningAndReady waits for numStatefulPods in ss to be Running and Ready.
func WaitForRunningAndReady(c clientset.Interface, numStatefulPods int32, ss *appsv1.StatefulSet) {
	WaitForRunning(c, numStatefulPods, numStatefulPods, ss)
}
