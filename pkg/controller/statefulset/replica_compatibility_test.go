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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package statefulset

import (
	"math"
	"reflect"
	"testing"
	"time"

	apps "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
)

type batchPodControl struct{ created, deleted, updated []*v1.Pod }

func (c *batchPodControl) CreateStatefulPod(_ *apps.StatefulSet, pod *v1.Pod) error {
	c.created = append(c.created, pod)
	return nil
}
func (c *batchPodControl) DeleteStatefulPod(_ *apps.StatefulSet, pod *v1.Pod) error {
	c.deleted = append(c.deleted, pod)
	return nil
}
func (c *batchPodControl) UpdateStatefulPod(_ *apps.StatefulSet, pod *v1.Pod) error {
	c.updated = append(c.updated, pod)
	return nil
}

func readyCompatibilityPod(set *apps.StatefulSet, ordinal int, revision string) *v1.Pod {
	pod := newStatefulSetPod(set, ordinal)
	pod.Status.Phase = v1.PodRunning
	pod.Status.Conditions = []v1.PodCondition{{Type: v1.PodReady, Status: v1.ConditionTrue}}
	setPodRevision(pod, revision)
	return pod
}

func TestHugeReplicaReconcileIsBounded(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		set := newStatefulSet(math.MaxInt32)
		if parallel {
			burst(set)
		}
		revision := newRevisionOrDie(set, 1)
		pc := &batchPodControl{}
		control := &defaultStatefulSetControl{podControl: pc}
		status, err := control.updateStatefulSet(set, revision, revision, 0, nil)
		wantCreated := 1
		var wantErr error
		if parallel {
			wantCreated, wantErr = maxPodOperationsPerSync, errReconcileLimit
		}
		if err != wantErr || len(pc.created) != wantCreated || status.Replicas != int32(wantCreated) {
			t.Fatalf("parallel %v: created %d, status %v, error %v", parallel, len(pc.created), status, err)
		}
		// Deletion must also avoid traversing or allocating the declared range.
		now := metav1.Now()
		set.DeletionTimestamp = &now
		pc.created = nil
		if _, err := control.updateStatefulSet(set, revision, revision, 0, nil); err != nil || len(pc.created) != 0 {
			t.Fatalf("deleting huge set: created %d, error %v", len(pc.created), err)
		}
	}
}

func TestReplicaBatchesConverge(t *testing.T) {
	set := burst(newStatefulSet(maxPodOperationsPerSync + 7))
	set.Annotations = map[string]string{helper.DeleteSlotsAnn: "[0,2]"}
	revision := newRevisionOrDie(set, 1)
	pc := &batchPodControl{}
	control := &defaultStatefulSetControl{podControl: pc}
	status, err := control.updateStatefulSet(set, revision, revision, 0, nil)
	if err != errReconcileLimit || len(pc.created) != maxPodOperationsPerSync || status.Replicas != maxPodOperationsPerSync {
		t.Fatalf("first batch: created %d, status %v, error %v", len(pc.created), status, err)
	}
	var pods []*v1.Pod
	for _, pod := range pc.created {
		pods = append(pods, readyCompatibilityPod(set, getOrdinal(pod), revision.Name))
	}
	pc.created = nil
	status, err = control.updateStatefulSet(set, revision, revision, 0, pods)
	if err != nil || len(pc.created) != 7 || status.Replicas != *set.Spec.Replicas {
		t.Fatalf("last batch: created %d, status %v, error %v", len(pc.created), status, err)
	}
	for _, pod := range pc.created {
		pods = append(pods, readyCompatibilityPod(set, getOrdinal(pod), revision.Name))
	}
	ordinals := helper.GetPodOrdinals(*set.Spec.Replicas, set)
	for _, pod := range pods {
		if !ordinals.Has(int32(getOrdinal(pod))) {
			t.Fatalf("unexpected Pod %s", pod.Name)
		}
		ordinals.Delete(int32(getOrdinal(pod)))
	}
	if ordinals.Len() != 0 {
		t.Fatalf("missing Pods %v", ordinals)
	}
	// Scale down uses the same operation budget and resumes at the next highest ordinal.
	*set.Spec.Replicas = 0
	pc.deleted = nil
	if _, err := control.updateStatefulSet(set, revision, revision, 0, pods); err != errReconcileLimit || len(pc.deleted) != maxPodOperationsPerSync {
		t.Fatalf("scale down batch: deleted %d, error %v", len(pc.deleted), err)
	}
	if getOrdinal(pc.deleted[0]) != maxPodOperationsPerSync+8 {
		t.Fatal("scale down did not start at highest ordinal")
	}
	pc.deleted = nil
	if _, err := control.updateStatefulSet(set, revision, revision, 0, pods[:7]); err != nil || len(pc.deleted) != 7 {
		t.Fatalf("last scale down batch: deleted %d, error %v", len(pc.deleted), err)
	}
}

func TestReplicaBudgetRequeuesAndRecordsProgress(t *testing.T) {
	set := burst(newStatefulSet(maxPodOperationsPerSync + 1))
	controller, pc := newFakeStatefulSetController(set)
	defer controller.queue.ShutDown()
	if err := pc.setsIndexer.Update(set); err != nil {
		t.Fatal(err)
	}
	if err := controller.syncStatefulSet(set, nil); err != nil {
		t.Fatal(err)
	}
	stored, err := pc.setsLister.StatefulSets(set.Namespace).Get(set.Name)
	if err != nil || stored.Status.Replicas != maxPodOperationsPerSync {
		t.Fatalf("batch progress not recorded: set %v, error %v", stored, err)
	}
	if err := wait.PollImmediate(10*time.Millisecond, time.Second, func() (bool, error) { return controller.queue.Len() > 0, nil }); err != nil {
		t.Fatal("unfinished batch was not requeued")
	}
	key, _ := controller.queue.Get()
	defer controller.queue.Done(key)
	if key != set.Namespace+"/"+set.Name {
		t.Fatalf("unexpected requeued key %v", key)
	}
}

// Model TiDB Operator v1's TiKVStsDesiredReplicas: configured replicas plus
// failure stores plus spare replicas during volume replacement. No Operator
// dependency is needed: these are the exact StatefulSet inputs it produces.
func TestTiDBOperatorFailoverAndVolumeReplacement(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		configured, failures, spare int32
	}{
		{"failover", 10000, 1, 0},
		{"volume-replacement", 10000, 0, 2},
		{"failover-during-volume-replacement", 10000, 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := burst(newStatefulSet(int(tc.configured + tc.failures + tc.spare)))
			set.Annotations = map[string]string{helper.DeleteSlotsAnn: "[0,2,2,2147483647]"}
			revision := newRevisionOrDie(set, 1)
			pc := &batchPodControl{}
			control := &defaultStatefulSetControl{podControl: pc}
			status, err := control.updateStatefulSet(set, revision, revision, 0, nil)
			if err != errReconcileLimit || status.Replicas != maxPodOperationsPerSync {
				t.Fatalf("status %v, error %v", status, err)
			}
			for _, pod := range pc.created {
				if getOrdinal(pod) == 0 || getOrdinal(pod) == 2 {
					t.Fatalf("created deleted slot %s", pod.Name)
				}
			}
		})
	}
	// Verify the complete small-cluster transition and recovery, including slots.
	set := burst(newStatefulSet(3))
	set.Annotations = map[string]string{helper.DeleteSlotsAnn: "[0,2]"}
	revision := newRevisionOrDie(set, 1)
	var pods []*v1.Pod
	for _, replicas := range []int32{3, 4, 6, 3} {
		*set.Spec.Replicas = replicas
		pc := &batchPodControl{}
		control := &defaultStatefulSetControl{podControl: pc}
		if _, err := control.updateStatefulSet(set, revision, revision, 0, pods); err != nil {
			t.Fatal(err)
		}
		deleted := map[string]bool{}
		for _, pod := range pc.deleted {
			deleted[pod.Name] = true
		}
		kept := []*v1.Pod{}
		for _, pod := range pods {
			if !deleted[pod.Name] {
				kept = append(kept, pod)
			}
		}
		for _, pod := range pc.created {
			kept = append(kept, readyCompatibilityPod(set, getOrdinal(pod), revision.Name))
		}
		pods = kept
		want := helper.GetPodOrdinals(replicas, set)
		if len(pods) != int(replicas) {
			t.Fatalf("got %d Pods, want %d", len(pods), replicas)
		}
		for _, pod := range pods {
			if !want.Has(int32(getOrdinal(pod))) {
				t.Fatalf("unexpected Pod %s", pod.Name)
			}
		}
	}
}

func TestSparseReplicaPartitionAndRecreation(t *testing.T) {
	for _, partition := range []int32{3, 5} {
		set := burst(newStatefulSet(3))
		set.Annotations = map[string]string{helper.DeleteSlotsAnn: "[0,2]"}
		current := newRevisionOrDie(set, 1)
		set.Spec.Template.Spec.Containers[0].Image = "new-image"
		update := newRevisionOrDie(set, 2)
		set.Spec.UpdateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{Partition: &partition}
		pods := []*v1.Pod{readyCompatibilityPod(set, 1, current.Name), readyCompatibilityPod(set, 3, current.Name), readyCompatibilityPod(set, 4, current.Name)}
		pc := &batchPodControl{}
		control := &defaultStatefulSetControl{podControl: pc, recorder: record.NewFakeRecorder(10)}
		if _, err := control.updateStatefulSet(set, current, update, 0, pods); err != nil {
			t.Fatal(err)
		}
		var got []int
		for _, pod := range pc.deleted {
			got = append(got, getOrdinal(pod))
		}
		var want []int
		if partition == 3 {
			want = []int{4}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("partition %d: deleted %v, want %v", partition, got, want)
		}
		// A failed Pod at ordinal 3 must be recreated at 3, not at slice index 1.
		pods[1].Status.Phase = v1.PodFailed
		pc.created, pc.deleted = nil, nil
		set.Spec.UpdateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
		if _, err := control.updateStatefulSet(set, current, update, 0, pods); err != nil {
			t.Fatal(err)
		}
		if len(pc.created) != 1 || getOrdinal(pc.created[0]) != 3 {
			t.Fatalf("wrong recreated Pods: %v", pc.created)
		}
	}
}

func TestTiDBOperatorHighOrdinalsWithDeleteSlots(t *testing.T) {
	set := burst(newStatefulSet(3))
	slots := sets.NewInt32()
	for i := int32(0); i <= 10000; i++ {
		slots.Insert(i)
	}
	if err := helper.SetDeleteSlots(set, slots); err != nil {
		t.Fatal(err)
	}
	revision := newRevisionOrDie(set, 1)
	pc := &batchPodControl{}
	control := &defaultStatefulSetControl{podControl: pc}
	status, err := control.updateStatefulSet(set, revision, revision, 0, nil)
	if err != nil || status.Replicas != 3 || len(pc.created) != 3 {
		t.Fatalf("status %v, created %d, error %v", status, len(pc.created), err)
	}
	for i, pod := range pc.created {
		if getOrdinal(pod) != 10001+i {
			t.Fatalf("unexpected ordinal %d", getOrdinal(pod))
		}
	}
}
