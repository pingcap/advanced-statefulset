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
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2esset "k8s.io/kubernetes/test/e2e/framework/statefulset"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	podutil "github.com/pingcap/advanced-statefulset/pkg/third_party/k8s"
	k8s "github.com/pingcap/advanced-statefulset/test/third_party/k8s"
)

const (
	// statefulSetPoll is a poll interval for StatefulSet tests
	statefulSetPoll = 10 * time.Second
	// statefulSetTimeout is a timeout interval for StatefulSet operations
	statefulSetTimeout = 10 * time.Minute
	// statefulPodTimeout is a timeout for stateful pods to change state
	statefulPodTimeout = 5 * time.Minute
)

var (
	statefulPodRegex = regexp.MustCompile("(.*)-([0-9]+)$")

	httpProbe = &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			HTTPGet: &v1.HTTPGetAction{
				Path: "/index.html",
				Port: intstr.IntOrString{IntVal: 80},
			},
		},
		PeriodSeconds:    1,
		SuccessThreshold: 1,
		FailureThreshold: 1,
	}
)

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
		k8s.Failf("StatefulSet %s/%s attempt to wait for partitioned update with updateStrategy %s",
			set.Namespace,
			set.Name,
			set.Spec.UpdateStrategy.Type)
	}
	if set.Spec.UpdateStrategy.RollingUpdate == nil || set.Spec.UpdateStrategy.RollingUpdate.Partition == nil {
		k8s.Failf("StatefulSet %s/%s attempt to wait for partitioned update with nil RollingUpdate or nil Partition",
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
			k8s.Logf("Waiting for StatefulSet %s/%s to complete update",
				set.Namespace,
				set.Name,
			)
			e2esset.SortStatefulPods(pods)
			for i := range pods.Items {
				if pods.Items[i].Labels[appsv1.StatefulSetRevisionLabel] != set.Status.UpdateRevision {
					k8s.Logf("Waiting for Pod %s/%s to have revision %s update revision %s",
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
				k8s.Logf("Waiting for Pod %s/%s to have revision %s update revision %s",
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
				k8s.Logf("Found %d stateful pods, waiting for %d", len(podList.Items), numPodsRunning)
				return false, nil
			}
			if int32(len(podList.Items)) > numPodsRunning {
				return false, fmt.Errorf("too many pods scheduled, expected %d got %d", numPodsRunning, len(podList.Items))
			}
			deleteSlots := helper.GetDeleteSlots(ss)
			replicaCount, deleteSlots := helper.GetMaxReplicaCountAndDeleteSlots(*ss.Spec.Replicas, deleteSlots)
			for _, p := range podList.Items {
				if deleteSlots.Has(int32(getStatefulPodOrdinal(&p))) {
					return false, fmt.Errorf("unexpected pod ordinal: %d for stateful set %q", getStatefulPodOrdinal(&p), ss.Name)
				}
				shouldBeReady := int32(getStatefulPodOrdinal(&p)) < replicaCount
				isReady := podutil.IsPodReady(&p)
				desiredReadiness := shouldBeReady == isReady
				k8s.Logf("Waiting for pod %v to enter %v - Ready=%v, currently %v - Ready=%v", p.Name, v1.PodRunning, shouldBeReady, p.Status.Phase, isReady)
				if p.Status.Phase != v1.PodRunning || !desiredReadiness {
					return false, nil
				}
			}
			return true, nil
		})
	if pollErr != nil {
		k8s.Failf("Failed waiting for pods to enter running: %v", pollErr)
	}
}

// WaitForRunningAndReady waits for numStatefulPods in ss to be Running and Ready.
func WaitForRunningAndReady(c clientset.Interface, numStatefulPods int32, ss *appsv1.StatefulSet) {
	WaitForRunning(c, numStatefulPods, numStatefulPods, ss)
}

type updateStatefulSetFunc func(*appsv1.StatefulSet)

// updateStatefulSetWithRetries updates statfulset template with retries.
func updateStatefulSetWithRetries(c clientset.Interface, namespace, name string, applyUpdate updateStatefulSetFunc) (statefulSet *appsv1.StatefulSet, err error) {
	statefulSets := c.AppsV1().StatefulSets(namespace)
	var updateErr error
	pollErr := wait.Poll(10*time.Millisecond, 1*time.Minute, func() (bool, error) {
		if statefulSet, err = statefulSets.Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
			return false, err
		}
		// Apply the update, then attempt to push it to the apiserver.
		applyUpdate(statefulSet)
		if statefulSet, err = statefulSets.Update(context.TODO(), statefulSet, metav1.UpdateOptions{}); err == nil {
			k8s.Logf("Updating stateful set %s", name)
			return true, nil
		}
		updateErr = err
		return false, nil
	})
	if pollErr == wait.ErrWaitTimeout {
		pollErr = fmt.Errorf("couldn't apply the provided updated to stateful set %q: %v", name, updateErr)
	}
	return statefulSet, pollErr
}

// setHTTPProbe sets the pod template's ReadinessProbe for Webserver StatefulSet containers.
// This probe can then be controlled with BreakHTTPProbe() and RestoreHTTPProbe().
// Note that this cannot be used together with PauseNewPods().
func setHTTPProbe(ss *appsv1.StatefulSet) {
	ss.Spec.Template.Spec.Containers[0].ReadinessProbe = httpProbe
}

// breakHTTPProbe breaks the readiness probe for Nginx StatefulSet containers in ss.
func breakHTTPProbe(c clientset.Interface, ss *appsv1.StatefulSet) error {
	path := httpProbe.HTTPGet.Path
	if path == "" {
		return fmt.Errorf("path expected to be not empty: %v", path)
	}
	// Ignore 'mv' errors to make this idempotent.
	cmd := fmt.Sprintf("mv -v /usr/local/apache2/htdocs%v /tmp/ || true", path)
	return e2esset.ExecInStatefulPods(c, ss, cmd)
}

// breakPodHTTPProbe breaks the readiness probe for Nginx StatefulSet containers in one pod.
func breakPodHTTPProbe(ss *appsv1.StatefulSet, pod *v1.Pod) error {
	path := httpProbe.HTTPGet.Path
	if path == "" {
		return fmt.Errorf("path expected to be not empty: %v", path)
	}
	// Ignore 'mv' errors to make this idempotent.
	cmd := fmt.Sprintf("mv -v /usr/local/apache2/htdocs%v /tmp/ || true", path)
	stdout, err := framework.RunHostCmdWithRetries(pod.Namespace, pod.Name, cmd, statefulSetPoll, statefulPodTimeout)
	k8s.Logf("stdout of %v on %v: %v", cmd, pod.Name, stdout)
	return err
}

// restoreHTTPProbe restores the readiness probe for Nginx StatefulSet containers in ss.
func restoreHTTPProbe(c clientset.Interface, ss *appsv1.StatefulSet) error {
	path := httpProbe.HTTPGet.Path
	if path == "" {
		return fmt.Errorf("path expected to be not empty: %v", path)
	}
	// Ignore 'mv' errors to make this idempotent.
	cmd := fmt.Sprintf("mv -v /tmp%v /usr/local/apache2/htdocs/ || true", path)
	return e2esset.ExecInStatefulPods(c, ss, cmd)
}

// restorePodHTTPProbe restores the readiness probe for Nginx StatefulSet containers in pod.
func restorePodHTTPProbe(ss *appsv1.StatefulSet, pod *v1.Pod) error {
	path := httpProbe.HTTPGet.Path
	if path == "" {
		return fmt.Errorf("path expected to be not empty: %v", path)
	}
	// Ignore 'mv' errors to make this idempotent.
	cmd := fmt.Sprintf("mv -v /tmp%v /usr/local/apache2/htdocs/ || true", path)
	stdout, err := framework.RunHostCmdWithRetries(pod.Namespace, pod.Name, cmd, statefulSetPoll, statefulPodTimeout)
	k8s.Logf("stdout of %v on %v: %v", cmd, pod.Name, stdout)
	return err
}

// getStatefulSet gets the StatefulSet named name in namespace.
func getStatefulSet(c clientset.Interface, namespace, name string) *appsv1.StatefulSet {
	ss, err := c.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		k8s.Failf("Failed to get StatefulSet %s/%s: %v", namespace, name, err)
	}
	return ss
}

// deleteStatefulPodAtIndex deletes the Pod with ordinal index in ss.
func deleteStatefulPodAtIndex(c clientset.Interface, index int, ss *appsv1.StatefulSet) {
	name := getStatefulSetPodNameAtIndex(index, ss)
	noGrace := int64(0)
	if err := c.CoreV1().Pods(ss.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{GracePeriodSeconds: &noGrace}); err != nil {
		k8s.Failf("Failed to delete stateful pod %v for StatefulSet %v/%v: %v", name, ss.Namespace, ss.Name, err)
	}
}

// getStatefulSetPodNameAtIndex gets formatted pod name given index.
func getStatefulSetPodNameAtIndex(index int, ss *appsv1.StatefulSet) string {
	// TODO: we won't use "-index" as the name strategy forever,
	// pull the name out from an identity mapper.
	return fmt.Sprintf("%v-%v", ss.Name, index)
}
