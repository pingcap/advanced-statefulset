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

// this file is copied from k8s.io/kubernetes/test/e2e/framework/pod/wait.go @v1.23.17

package pod

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"text/tabwriter"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/util/podutils"

	"github.com/pingcap/advanced-statefulset/test/third_party/k8s"
	"github.com/pingcap/advanced-statefulset/test/third_party/k8s/utils"
)

const (
	// poll is how often to poll pods, nodes and claims.
	poll = 2 * time.Second
)

// errorBadPodsStates create error message of basic info of bad pods for debugging.
func errorBadPodsStates(badPods []v1.Pod, desiredPods int, ns, desiredState string, timeout time.Duration, err error) string {
	errStr := fmt.Sprintf("%d / %d pods in namespace %q are NOT in %s state in %v\n", len(badPods), desiredPods, ns, desiredState, timeout)
	if err != nil {
		errStr += fmt.Sprintf("Last error: %s\n", err)
	}
	// Print bad pods info only if there are fewer than 10 bad pods
	if len(badPods) > 10 {
		return errStr + "There are too many bad pods. Please check log for details."
	}

	buf := bytes.NewBuffer(nil)
	w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "POD\tNODE\tPHASE\tGRACE\tCONDITIONS")
	for _, badPod := range badPods {
		grace := ""
		if badPod.DeletionGracePeriodSeconds != nil {
			grace = fmt.Sprintf("%ds", *badPod.DeletionGracePeriodSeconds)
		}
		podInfo := fmt.Sprintf("%s\t%s\t%s\t%s\t%+v",
			badPod.ObjectMeta.Name, badPod.Spec.NodeName, badPod.Status.Phase, grace, badPod.Status.Conditions)
		fmt.Fprintln(w, podInfo)
	}
	w.Flush()
	return errStr + buf.String()
}

// WaitForPodsRunningReady waits up to timeout to ensure that all pods in
// namespace ns are either running and ready, or failed but controlled by a
// controller. Also, it ensures that at least minPods are running and
// ready. It has separate behavior from other 'wait for' pods functions in
// that it requests the list of pods on every iteration. This is useful, for
// example, in cluster startup, because the number of pods increases while
// waiting. All pods that are in SUCCESS state are not counted.
//
// If ignoreLabels is not empty, pods matching this selector are ignored.
//
// If minPods or allowedNotReadyPods are -1, this method returns immediately
// without waiting.
func WaitForPodsRunningReady(c clientset.Interface, ns string, minPods, allowedNotReadyPods int32, timeout time.Duration, ignoreLabels map[string]string) error {
	if minPods == -1 || allowedNotReadyPods == -1 {
		return nil
	}

	ignoreSelector := labels.SelectorFromSet(map[string]string{})
	start := time.Now()
	k8s.Logf("Waiting up to %v for all pods (need at least %d) in namespace '%s' to be running and ready",
		timeout, minPods, ns)
	var ignoreNotReady bool
	badPods := []v1.Pod{}
	desiredPods := 0
	notReady := int32(0)
	var lastAPIError error

	if wait.PollImmediate(poll, timeout, func() (bool, error) {
		// We get the new list of pods, replication controllers, and
		// replica sets in every iteration because more pods come
		// online during startup and we want to ensure they are also
		// checked.
		replicas, replicaOk := int32(0), int32(0)
		// Clear API error from the last attempt in case the following calls succeed.
		lastAPIError = nil

		rcList, err := c.CoreV1().ReplicationControllers(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			k8s.Logf("Error getting replication controllers in namespace '%s': %v", ns, err)
			lastAPIError = err
			return false, err
		}
		for _, rc := range rcList.Items {
			replicas += *rc.Spec.Replicas
			replicaOk += rc.Status.ReadyReplicas
		}

		rsList, err := c.AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			lastAPIError = err
			k8s.Logf("Error getting replication sets in namespace %q: %v", ns, err)
			return false, err
		}
		for _, rs := range rsList.Items {
			replicas += *rs.Spec.Replicas
			replicaOk += rs.Status.ReadyReplicas
		}

		podList, err := c.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			lastAPIError = err
			k8s.Logf("Error getting pods in namespace '%s': %v", ns, err)
			return false, err
		}
		nOk := int32(0)
		notReady = int32(0)
		badPods = []v1.Pod{}
		desiredPods = len(podList.Items)
		for _, pod := range podList.Items {
			if len(ignoreLabels) != 0 && ignoreSelector.Matches(labels.Set(pod.Labels)) {
				continue
			}
			res, err := utils.PodRunningReady(&pod)
			switch {
			case res && err == nil:
				nOk++
			case pod.Status.Phase == v1.PodSucceeded:
				k8s.Logf("The status of Pod %s is Succeeded, skipping waiting", pod.ObjectMeta.Name)
				// it doesn't make sense to wait for this pod
				continue
			case pod.Status.Phase != v1.PodFailed:
				k8s.Logf("The status of Pod %s is %s (Ready = false), waiting for it to be either Running (with Ready = true) or Failed", pod.ObjectMeta.Name, pod.Status.Phase)
				notReady++
				badPods = append(badPods, pod)
			default:
				if metav1.GetControllerOf(&pod) == nil {
					k8s.Logf("Pod %s is Failed, but it's not controlled by a controller", pod.ObjectMeta.Name)
					badPods = append(badPods, pod)
				}
				//ignore failed pods that are controlled by some controller
			}
		}

		k8s.Logf("%d / %d pods in namespace '%s' are running and ready (%d seconds elapsed)",
			nOk, len(podList.Items), ns, int(time.Since(start).Seconds()))
		k8s.Logf("expected %d pod replicas in namespace '%s', %d are Running and Ready.", replicas, ns, replicaOk)

		if replicaOk == replicas && nOk >= minPods && len(badPods) == 0 {
			return true, nil
		}
		ignoreNotReady = (notReady <= allowedNotReadyPods)
		LogPodStates(badPods)
		return false, nil
	}) != nil {
		if !ignoreNotReady {
			return errors.New(errorBadPodsStates(badPods, desiredPods, ns, "RUNNING and READY", timeout, lastAPIError))
		}
		k8s.Logf("Number of not-ready pods (%d) is below the allowed threshold (%d).", notReady, allowedNotReadyPods)
	}
	return nil
}

// WaitForPodCondition waits a pods to be matched to the given condition.
func WaitForPodCondition(c clientset.Interface, ns, podName, desc string, timeout time.Duration, condition podCondition) error {
	k8s.Logf("Waiting up to %v for pod %q in namespace %q to be %q", timeout, podName, ns, desc)
	var lastPodError error
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		pod, err := c.CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
		lastPodError = err
		if err != nil {
			if apierrors.IsNotFound(err) {
				k8s.Logf("Pod %q in namespace %q not found. Error: %v", podName, ns, err)
			} else {
				k8s.Logf("Get pod %q in namespace %q failed, ignoring for %v. Error: %v", podName, ns, poll, err)
			}
			continue
		}
		// log now so that current pod info is reported before calling `condition()`
		k8s.Logf("Pod %q: Phase=%q, Reason=%q, readiness=%t. Elapsed: %v",
			podName, pod.Status.Phase, pod.Status.Reason, podutils.IsPodReady(pod), time.Since(start))
		if done, err := condition(pod); done {
			if err == nil {
				k8s.Logf("Pod %q satisfied condition %q", podName, desc)
			}
			return err
		}
	}
	if apierrors.IsNotFound(lastPodError) {
		// return for compatbility with other functions testing for IsNotFound
		return lastPodError
	}
	return fmt.Errorf("Gave up after waiting %v for pod %q to be %q", timeout, podName, desc)
}
