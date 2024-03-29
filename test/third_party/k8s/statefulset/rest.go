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

// this file is copied from k8s.io/kubernetes/test/e2e/framework/statefulset/rest.go @v1.23.17

package statefulset

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	podutils "github.com/pingcap/advanced-statefulset/pkg/third_party/k8s"
	"github.com/pingcap/advanced-statefulset/test/third_party/k8s"
	e2emanifest "github.com/pingcap/advanced-statefulset/test/third_party/k8s/manifest"
)

// CreateStatefulSet creates a StatefulSet from the manifest at manifestPath in the Namespace ns using kubectl create.
func CreateStatefulSet(c clientset.Interface, manifestPath, ns string) *appsv1.StatefulSet {
	mkpath := func(file string) string {
		return filepath.Join(manifestPath, file)
	}

	k8s.Logf("Parsing statefulset from %v", mkpath("statefulset.yaml"))
	ss, err := e2emanifest.StatefulSetFromManifest(mkpath("statefulset.yaml"), ns)
	k8s.ExpectNoError(err)
	k8s.Logf("Parsing service from %v", mkpath("service.yaml"))
	svc, err := e2emanifest.SvcFromManifest(mkpath("service.yaml"))
	k8s.ExpectNoError(err)

	k8s.Logf(fmt.Sprintf("creating " + ss.Name + " service"))
	_, err = c.CoreV1().Services(ns).Create(context.TODO(), svc, metav1.CreateOptions{})
	k8s.ExpectNoError(err)

	k8s.Logf(fmt.Sprintf("creating statefulset %v/%v with %d replicas and selector %+v", ss.Namespace, ss.Name, *(ss.Spec.Replicas), ss.Spec.Selector))
	_, err = c.AppsV1().StatefulSets(ns).Create(context.TODO(), ss, metav1.CreateOptions{})
	k8s.ExpectNoError(err)
	WaitForRunningAndReady(c, *ss.Spec.Replicas, ss)
	return ss
}

// GetPodList gets the current Pods in ss.
func GetPodList(c clientset.Interface, ss *appsv1.StatefulSet) *v1.PodList {
	selector, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector)
	k8s.ExpectNoError(err)
	podList, err := c.CoreV1().Pods(ss.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	k8s.ExpectNoError(err)
	return podList
}

// DeleteAllStatefulSets deletes all StatefulSet API Objects in Namespace ns.
func DeleteAllStatefulSets(c clientset.Interface, ns string) {
	ssList, err := c.AppsV1().StatefulSets(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labels.Everything().String()})
	k8s.ExpectNoError(err)

	// Scale down each statefulset, then delete it completely.
	// Deleting a pvc without doing this will leak volumes, #25101.
	errList := []string{}
	for i := range ssList.Items {
		ss := &ssList.Items[i]
		var err error
		if ss, err = Scale(c, ss, 0); err != nil {
			errList = append(errList, fmt.Sprintf("%v", err))
		}
		WaitForStatusReplicas(c, ss, 0)
		k8s.Logf("Deleting statefulset %v", ss.Name)
		// Use OrphanDependents=false so it's deleted synchronously.
		// We already made sure the Pods are gone inside Scale().
		if err := c.AppsV1().StatefulSets(ss.Namespace).Delete(context.TODO(), ss.Name, metav1.DeleteOptions{OrphanDependents: new(bool)}); err != nil {
			errList = append(errList, fmt.Sprintf("%v", err))
		}
	}

	// pvs are global, so we need to wait for the exact ones bound to the statefulset pvcs.
	pvNames := sets.NewString()
	// TODO: Don't assume all pvcs in the ns belong to a statefulset
	pvcPollErr := wait.PollImmediate(StatefulSetPoll, StatefulSetTimeout, func() (bool, error) {
		pvcList, err := c.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labels.Everything().String()})
		if err != nil {
			k8s.Logf("WARNING: Failed to list pvcs, retrying %v", err)
			return false, nil
		}
		for _, pvc := range pvcList.Items {
			pvNames.Insert(pvc.Spec.VolumeName)
			// TODO: Double check that there are no pods referencing the pvc
			k8s.Logf("Deleting pvc: %v with volume %v", pvc.Name, pvc.Spec.VolumeName)
			if err := c.CoreV1().PersistentVolumeClaims(ns).Delete(context.TODO(), pvc.Name, metav1.DeleteOptions{}); err != nil {
				return false, nil
			}
		}
		return true, nil
	})
	if pvcPollErr != nil {
		errList = append(errList, fmt.Sprintf("Timeout waiting for pvc deletion."))
	}

	pollErr := wait.PollImmediate(StatefulSetPoll, StatefulSetTimeout, func() (bool, error) {
		pvList, err := c.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{LabelSelector: labels.Everything().String()})
		if err != nil {
			k8s.Logf("WARNING: Failed to list pvs, retrying %v", err)
			return false, nil
		}
		waitingFor := []string{}
		for _, pv := range pvList.Items {
			if pvNames.Has(pv.Name) {
				waitingFor = append(waitingFor, fmt.Sprintf("%v: %+v", pv.Name, pv.Status))
			}
		}
		if len(waitingFor) == 0 {
			return true, nil
		}
		k8s.Logf("Still waiting for pvs of statefulset to disappear:\n%v", strings.Join(waitingFor, "\n"))
		return false, nil
	})
	if pollErr != nil {
		errList = append(errList, fmt.Sprintf("Timeout waiting for pv provisioner to delete pvs, this might mean the test leaked pvs."))
	}
	if len(errList) != 0 {
		k8s.ExpectNoError(fmt.Errorf("%v", strings.Join(errList, "\n")))
	}
}

// Scale scales ss to count replicas.
func Scale(c clientset.Interface, ss *appsv1.StatefulSet, count int32) (*appsv1.StatefulSet, error) {
	name := ss.Name
	ns := ss.Namespace

	k8s.Logf("Scaling statefulset %s to %d", name, count)
	ss = update(c, ns, name, func(ss *appsv1.StatefulSet) { *(ss.Spec.Replicas) = count })

	var statefulPodList *v1.PodList
	pollErr := wait.PollImmediate(StatefulSetPoll, StatefulSetTimeout, func() (bool, error) {
		statefulPodList = GetPodList(c, ss)
		if int32(len(statefulPodList.Items)) == count {
			return true, nil
		}
		return false, nil
	})
	if pollErr != nil {
		unhealthy := []string{}
		for _, statefulPod := range statefulPodList.Items {
			delTs, phase, readiness := statefulPod.DeletionTimestamp, statefulPod.Status.Phase, podutils.IsPodReady(&statefulPod)
			if delTs != nil || phase != v1.PodRunning || !readiness {
				unhealthy = append(unhealthy, fmt.Sprintf("%v: deletion %v, phase %v, readiness %v", statefulPod.Name, delTs, phase, readiness))
			}
		}
		return ss, fmt.Errorf("Failed to scale statefulset to %d in %v. Remaining pods:\n%v", count, StatefulSetTimeout, unhealthy)
	}
	return ss, nil
}

// udpate updates a statefulset, and it is only used within rest.go
func update(c clientset.Interface, ns, name string, update func(ss *appsv1.StatefulSet)) *appsv1.StatefulSet {
	for i := 0; i < 3; i++ {
		ss, err := c.AppsV1().StatefulSets(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			k8s.Failf("failed to get statefulset %q: %v", name, err)
		}
		update(ss)
		ss, err = c.AppsV1().StatefulSets(ns).Update(context.TODO(), ss, metav1.UpdateOptions{})
		if err == nil {
			return ss
		}
		if !apierrors.IsConflict(err) && !apierrors.IsServerTimeout(err) {
			k8s.Failf("failed to update statefulset %q: %v", name, err)
		}
	}
	k8s.Failf("too many retries draining statefulset %q", name)
	return nil
}

// UpdateReplicas updates the replicas of ss to count.
func UpdateReplicas(c clientset.Interface, ss *appsv1.StatefulSet, count int32) {
	update(c, ss.Namespace, ss.Name, func(ss *appsv1.StatefulSet) { *(ss.Spec.Replicas) = count })
}

// Restart scales ss to 0 and then back to its previous number of replicas.
func Restart(c clientset.Interface, ss *appsv1.StatefulSet) {
	oldReplicas := *(ss.Spec.Replicas)
	ss, err := Scale(c, ss, 0)
	k8s.ExpectNoError(err)
	// Wait for controller to report the desired number of Pods.
	// This way we know the controller has observed all Pod deletions
	// before we scale it back up.
	WaitForStatusReplicas(c, ss, 0)
	update(c, ss.Namespace, ss.Name, func(ss *appsv1.StatefulSet) { *(ss.Spec.Replicas) = oldReplicas })
}

// CheckHostname verifies that all Pods in ss have the correct Hostname. If the returned error is not nil than verification failed.
func CheckHostname(c clientset.Interface, ss *appsv1.StatefulSet) error {
	cmd := "printf $(hostname)"
	podList := GetPodList(c, ss)
	for _, statefulPod := range podList.Items {
		hostname, err := k8s.RunHostCmdWithRetries(statefulPod.Namespace, statefulPod.Name, cmd, StatefulSetPoll, StatefulPodTimeout)
		if err != nil {
			return err
		}
		if hostname != statefulPod.Name {
			return fmt.Errorf("unexpected hostname (%s) and stateful pod name (%s) not equal", hostname, statefulPod.Name)
		}
	}
	return nil
}

// CheckMount checks that the mount at mountPath is valid for all Pods in ss.
func CheckMount(c clientset.Interface, ss *appsv1.StatefulSet, mountPath string) error {
	for _, cmd := range []string{
		// Print inode, size etc
		fmt.Sprintf("ls -idlh %v", mountPath),
		// Print subdirs
		fmt.Sprintf("find %v", mountPath),
		// Try writing
		fmt.Sprintf("touch %v", filepath.Join(mountPath, fmt.Sprintf("%v", time.Now().UnixNano()))),
	} {
		if err := ExecInStatefulPods(c, ss, cmd); err != nil {
			return fmt.Errorf("failed to execute %v, error: %v", cmd, err)
		}
	}
	return nil
}

// CheckServiceName asserts that the ServiceName for ss is equivalent to expectedServiceName.
func CheckServiceName(ss *appsv1.StatefulSet, expectedServiceName string) error {
	k8s.Logf("Checking if statefulset spec.serviceName is %s", expectedServiceName)

	if expectedServiceName != ss.Spec.ServiceName {
		return fmt.Errorf("wrong service name governing statefulset. Expected %s got %s",
			expectedServiceName, ss.Spec.ServiceName)
	}

	return nil
}

// ExecInStatefulPods executes cmd in all Pods in ss. If a error occurs it is returned and cmd is not execute in any subsequent Pods.
func ExecInStatefulPods(c clientset.Interface, ss *appsv1.StatefulSet, cmd string) error {
	podList := GetPodList(c, ss)
	for _, statefulPod := range podList.Items {
		stdout, err := k8s.RunHostCmdWithRetries(statefulPod.Namespace, statefulPod.Name, cmd, StatefulSetPoll, StatefulPodTimeout)
		k8s.Logf("stdout of %v on %v: %v", cmd, statefulPod.Name, stdout)
		if err != nil {
			return err
		}
	}
	return nil
}
