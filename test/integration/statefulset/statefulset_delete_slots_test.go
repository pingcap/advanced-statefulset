/*
Copyright 2018 The Kubernetes Authors.

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

package statefulset

import (
	"fmt"
	"testing"

	appsv1alpha1 "github.com/cofyc/advanced-statefulset/pkg/apis/apps/v1alpha1"
	integrationutil "github.com/cofyc/advanced-statefulset/test/integration/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

func TestDeletedSlots(t *testing.T) {
	closeFn, rm, informers, c, appsinformers, pcc := scSetup(t)
	defer closeFn()
	ns := integrationutil.CreateTestingNamespace("test-spec-replicas-change", c, t)
	defer integrationutil.DeleteTestingNamespace(ns, c, t)
	stopCh := runControllerAndInformers(rm, informers, appsinformers)
	defer close(stopCh)

	createHeadlessService(t, c, newHeadlessService(ns.Name))
	sts := newSTS("sts", ns.Name, 2)
	stss, _ := createSTSsPods(t, c, pcc, []*appsv1alpha1.StatefulSet{sts}, []*v1.Pod{})
	sts = stss[0]
	waitSTSStable(t, pcc, sts)

	// Update .Spec.Replicas and verify .Status.Replicas is changed accordingly
	scaleSTS(t, pcc, sts, 3)
	scaleSTS(t, pcc, sts, 0)
	scaleSTS(t, pcc, sts, 2)

	scaleSTS(t, pcc, sts, 4) // 0, 1, 2, 3
	scaleSTS(t, pcc, sts, 3) // 0, 1, 2

	scaleInSTSByDeletingSlots(t, pcc, sts, 1)
	checkPodIdentifiers(t, c, sts, 0, 2)

	t.Logf(fmt.Sprintf("scale to replicas %d with delete slots %v", 0, []int{1}))
	scaleSTSWithDeletedSlots(t, pcc, sts, 0, sets.NewInt(1))
	checkPodIdentifiers(t, c, sts)

	t.Logf(fmt.Sprintf("scale to replicas %d with delete slots %v", 4, []int{}))
	scaleSTSWithDeletedSlots(t, pcc, sts, 4, sets.NewInt())
	checkPodIdentifiers(t, c, sts, 0, 1, 2, 3)

	t.Logf(fmt.Sprintf("scale to replicas %d with delete slots %v", 3, []int{0}))
	scaleSTSWithDeletedSlots(t, pcc, sts, 3, sets.NewInt(0))
	checkPodIdentifiers(t, c, sts, 1, 2, 3)

	// Add a template annotation change to test STS's status does update
	// without .Spec.Replicas change
	stsClient := pcc.AppsV1alpha1().StatefulSets(ns.Name)
	var oldGeneration int64
	newSTS := updateSTS(t, stsClient, sts.Name, func(sts *appsv1alpha1.StatefulSet) {
		oldGeneration = sts.Generation
		klog.Infof("annotations: %+v", sts.Annotations)
		sts.Spec.Template.Annotations = map[string]string{"test": "annotation"}
	})
	savedGeneration := newSTS.Generation
	if savedGeneration == oldGeneration {
		t.Fatalf("failed to verify .Generation has incremented for sts %s", sts.Name)
	}

	if err := wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		newSTS, err := stsClient.Get(sts.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return newSTS.Status.ObservedGeneration >= savedGeneration, nil
	}); err != nil {
		t.Fatalf("failed to verify .Status.ObservedGeneration has incremented for sts %s: %v", sts.Name, err)
	}
}
