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

// this file is copied from k8s.io/kubernetes/test/e2e/framework/statefulset/wait.go @v1.23.17

package statefulset

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/util/podutils"

	"github.com/pingcap/advanced-statefulset/test/third_party/k8s"
)

// WaitForRunning waits for numPodsRunning in ss to be Running and for the first
// numPodsReady ordinals to be Ready.
func WaitForRunning(c clientset.Interface, numPodsRunning, numPodsReady int32, ss *appsv1.StatefulSet) {
	pollErr := wait.PollImmediate(StatefulSetPoll, StatefulSetTimeout,
		func() (bool, error) {
			podList := GetPodList(c, ss)
			SortStatefulPods(podList)
			if int32(len(podList.Items)) < numPodsRunning {
				k8s.Logf("Found %d stateful pods, waiting for %d", len(podList.Items), numPodsRunning)
				return false, nil
			}
			if int32(len(podList.Items)) > numPodsRunning {
				return false, fmt.Errorf("too many pods scheduled, expected %d got %d", numPodsRunning, len(podList.Items))
			}
			for _, p := range podList.Items {
				shouldBeReady := getStatefulPodOrdinal(&p) < int(numPodsReady)
				isReady := podutils.IsPodReady(&p)
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

// WaitForState periodically polls for the ss and its pods until the until function returns either true or an error
func WaitForState(c clientset.Interface, ss *appsv1.StatefulSet, until func(*appsv1.StatefulSet, *v1.PodList) (bool, error)) {
	pollErr := wait.PollImmediate(StatefulSetPoll, StatefulSetTimeout,
		func() (bool, error) {
			ssGet, err := c.AppsV1().StatefulSets(ss.Namespace).Get(context.TODO(), ss.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			podList := GetPodList(c, ssGet)
			return until(ssGet, podList)
		})
	if pollErr != nil {
		k8s.Failf("Failed waiting for state update: %v", pollErr)
	}
}

// WaitForRunningAndReady waits for numStatefulPods in ss to be Running and Ready.
func WaitForRunningAndReady(c clientset.Interface, numStatefulPods int32, ss *appsv1.StatefulSet) {
	WaitForRunning(c, numStatefulPods, numStatefulPods, ss)
}

// WaitForStatusReplicas waits for the ss.Status.Replicas to be equal to expectedReplicas
func WaitForStatusReplicas(c clientset.Interface, ss *appsv1.StatefulSet, expectedReplicas int32) {
	k8s.Logf("Waiting for statefulset status.replicas updated to %d", expectedReplicas)

	ns, name := ss.Namespace, ss.Name
	pollErr := wait.PollImmediate(StatefulSetPoll, StatefulSetTimeout,
		func() (bool, error) {
			ssGet, err := c.AppsV1().StatefulSets(ns).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if ssGet.Status.ObservedGeneration < ss.Generation {
				return false, nil
			}
			if ssGet.Status.Replicas != expectedReplicas {
				k8s.Logf("Waiting for stateful set status.replicas to become %d, currently %d", expectedReplicas, ssGet.Status.Replicas)
				return false, nil
			}
			return true, nil
		})
	if pollErr != nil {
		k8s.Failf("Failed waiting for stateful set status.replicas updated to %d: %v", expectedReplicas, pollErr)
	}
}

// WaitForPodReady waits for the Pod named podName in set to exist and have a Ready condition.
func WaitForPodReady(c clientset.Interface, set *appsv1.StatefulSet, podName string) (*appsv1.StatefulSet, *v1.PodList) {
	var pods *v1.PodList
	WaitForState(c, set, func(set2 *appsv1.StatefulSet, pods2 *v1.PodList) (bool, error) {
		set = set2
		pods = pods2
		for i := range pods.Items {
			if pods.Items[i].Name == podName {
				return podutils.IsPodReady(&pods.Items[i]), nil
			}
		}
		return false, nil
	})
	return set, pods
}

// WaitForStatusReadyReplicas waits for the ss.Status.ReadyReplicas to be equal to expectedReplicas
func WaitForStatusReadyReplicas(c clientset.Interface, ss *appsv1.StatefulSet, expectedReplicas int32) {
	k8s.Logf("Waiting for statefulset status.replicas updated to %d", expectedReplicas)

	ns, name := ss.Namespace, ss.Name
	pollErr := wait.PollImmediate(StatefulSetPoll, StatefulSetTimeout,
		func() (bool, error) {
			ssGet, err := c.AppsV1().StatefulSets(ns).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if ssGet.Status.ObservedGeneration < ss.Generation {
				return false, nil
			}
			if ssGet.Status.ReadyReplicas != expectedReplicas {
				k8s.Logf("Waiting for stateful set status.readyReplicas to become %d, currently %d", expectedReplicas, ssGet.Status.ReadyReplicas)
				return false, nil
			}
			return true, nil
		})
	if pollErr != nil {
		k8s.Failf("Failed waiting for stateful set status.readyReplicas updated to %d: %v", expectedReplicas, pollErr)
	}
}

// Saturate waits for all Pods in ss to become Running and Ready.
func Saturate(c clientset.Interface, ss *appsv1.StatefulSet) {
	var i int32
	for i = 0; i < *(ss.Spec.Replicas); i++ {
		k8s.Logf("Waiting for stateful pod at index %v to enter Running", i)
		WaitForRunning(c, i+1, i, ss)
		k8s.Logf("Resuming stateful pod at index %v", i)
		ResumeNextPod(c, ss)
	}
}
