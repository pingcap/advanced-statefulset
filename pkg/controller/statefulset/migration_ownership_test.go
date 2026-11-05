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

package statefulset

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	kubeapps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestMigrationWaitsForOrphaningWithoutVersionBinding(t *testing.T) {
	for _, metadata := range []string{"old-helper", "new-helper", "operator-removed-annotation"} {
		t.Run(metadata, func(t *testing.T) {
			set := newStatefulSet(1)
			set.UID = "asts-uid"
			set.Status.CurrentRevision = "current"
			set.Status.UpdateRevision = "update"
			revisions := []*kubeapps.ControllerRevision{}
			for i, name := range []string{"history", "current", "update"} {
				versionSet := set.DeepCopy()
				versionSet.Spec.Template.Spec.Containers[0].Image = name
				revision := newRevisionOrDie(versionSet, int64(i))
				revision.Name = name
				markRevisionForUpgrade(set, revision)
				revision.OwnerReferences[0].APIVersion = kubeapps.SchemeGroupVersion.String()
				revision.OwnerReferences[0].UID = "source-uid"
				revision.Annotations = nil
				if metadata != "old-helper" {
					revision.Annotations = map[string]string{helper.UpgradeFromStatefulSetUIDAnn: "source-uid"}
				}
				revisions = append(revisions, revision)
			}
			set.Annotations = nil
			if metadata == "new-helper" {
				set.Annotations = map[string]string{helper.UpgradeFromStatefulSetUIDAnn: "source-uid"}
			}
			kube := fake.NewSimpleClientset(revisions[0], revisions[1], revisions[2])
			control := &defaultStatefulSetControl{csAppsV1: kube.AppsV1()}
			// Even matching optional metadata must not authorize using another owner's data.
			if _, err := control.ListRevisions(set); err == nil {
				t.Fatal("migration must wait until the source owner is removed")
			}
			for _, action := range kube.Actions() {
				if action.GetVerb() != "list" {
					t.Fatalf("source-owned history caused API mutation: %v", action)
				}
			}
			// Model GC's orphan propagation from the old helper's source deletion.
			for _, revision := range revisions {
				revision.OwnerReferences = nil
				if _, err := kube.AppsV1().ControllerRevisions(set.Namespace).Update(context.Background(), revision, metav1.UpdateOptions{}); err != nil {
					t.Fatal(err)
				}
			}
			got, err := control.ListRevisions(set)
			if err != nil || len(got) != 3 {
				t.Fatalf("migration lost rollback history: got %d revisions, error %v", len(got), err)
			}
			if err := control.AdoptOrphanRevisions(set, got); err != nil {
				t.Fatal(err)
			}
			for _, revision := range got {
				if _, err := syncLabels(kube, set, revision); err != nil {
					t.Fatal(err)
				}
			}
			// Operator reconciliation can replace all annotations without losing history.
			set.Annotations = nil
			got, err = control.ListRevisions(set)
			if err != nil || len(got) != 3 {
				t.Fatalf("adopted history lost: got %d revisions, error %v", len(got), err)
			}
			var history *kubeapps.ControllerRevision
			for _, revision := range got {
				owner := metav1.GetControllerOf(revision)
				if owner == nil || owner.UID != set.UID {
					t.Fatal("history not adopted by the ASTS")
				}
				if shouldSyncLabels(revision) {
					t.Fatal("migration marker not cleared")
				}
				if revision.Name == "history" {
					history = revision
				}
			}
			if history == nil {
				t.Fatal("old rollback history missing")
			}
			restored, err := ApplyRevision(set, history)
			if err != nil || restored.Spec.Template.Spec.Containers[0].Image != "history" {
				t.Fatalf("rollback payload not preserved: %v", err)
			}
			_, rollback, _, err := control.getStatefulSetRevisions(restored, got)
			if err != nil || rollback.Name != history.Name {
				t.Fatalf("adopted history unavailable for rollback: %v", err)
			}

		})
	}
}

func TestReferencedForeignRevisionWaitsWithoutWrites(t *testing.T) {
	set := newStatefulSet(1)
	revision := newRevisionOrDie(set, 1)
	revision.OwnerReferences[0].UID = "foreign"
	set.Status.CurrentRevision = revision.Name
	kube := fake.NewSimpleClientset(revision)
	control := &defaultStatefulSetControl{csAppsV1: kube.AppsV1()}
	if err := control.UpdateStatefulSet(set, nil); err == nil {
		t.Fatal("foreign current history must not fall back to a new revision")
	}
	for _, action := range kube.Actions() {
		if action.GetVerb() != "list" {
			t.Fatalf("foreign history caused API mutation: %v", action)
		}
	}
}

func TestLegacyOrphanMarkerAndEmptySelector(t *testing.T) {
	set := newStatefulSet(1)
	revision := newRevisionOrDie(set, 1)
	revision.OwnerReferences = nil
	markRevisionForUpgrade(set, revision)
	set.Annotations = nil
	revision.Annotations = nil
	kube := fake.NewSimpleClientset(revision)
	control := &defaultStatefulSetControl{csAppsV1: kube.AppsV1()}
	got, err := control.ListRevisions(set)
	if err != nil || len(got) != 1 {
		t.Fatalf("legacy orphan marker rejected: %v %v", got, err)
	}
	set.Spec.Selector = &metav1.LabelSelector{}
	kube.ClearActions()
	if _, err := control.ListRevisions(set); err == nil {
		t.Fatal("empty selector accepted")
	}
	if len(kube.Actions()) != 0 {
		t.Fatal("empty selector listed namespace history")
	}
}

func TestUnlistedForeignStatusRevisionBlocksSync(t *testing.T) {
	for _, reference := range []string{"current", "update", "both"} {
		t.Run(reference, func(t *testing.T) {
			set := newStatefulSet(1)
			previous := set.DeepCopy()
			previous.Spec.Template.Spec.Containers[0].Image = "foreign-template"
			foreign := newRevisionOrDie(previous, 1)
			foreign.OwnerReferences[0].UID = "other-controller"
			foreign.Labels = nil // Neither the selector nor the migration label LIST can find it.
			if reference != "update" {
				set.Status.CurrentRevision = foreign.Name
			}
			if reference != "current" {
				set.Status.UpdateRevision = foreign.Name
			}
			set.Status.Replicas = 1
			set.Status.ReadyReplicas = 1
			set.Status.CurrentReplicas = 1
			set.Status.UpdatedReplicas = 1
			originalStatus := set.DeepCopy().Status
			pod := newStatefulSetPod(previous, 0)
			pod.Status.Phase = v1.PodRunning
			pod.Status.Conditions = []v1.PodCondition{{Type: v1.PodReady, Status: v1.ConditionTrue}}
			setPodRevision(pod, foreign.Name)
			controller, control := newFakeStatefulSetController(set, foreign)
			defer controller.queue.ShutDown()
			if err := control.setsIndexer.Add(set); err != nil {
				t.Fatal(err)
			}
			if err := control.podsIndexer.Add(pod); err != nil {
				t.Fatal(err)
			}
			if err := controller.sync(set.Namespace + "/" + set.Name); err == nil {
				t.Fatal("unlisted foreign history must block reconciliation")
			}
			got, err := control.setsLister.StatefulSets(set.Namespace).Get(set.Name)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Status, originalStatus) {
				t.Fatal("status changed while waiting for foreign history")
			}
			gotPod, err := control.podsLister.Pods(set.Namespace).Get(pod.Name)
			if err != nil || !reflect.DeepEqual(gotPod, pod) {
				t.Fatalf("existing Pod changed: %v", err)
			}
			for _, action := range controller.kubeClient.(*fake.Clientset).Actions() {
				if action.GetVerb() != "list" && action.GetVerb() != "get" {
					t.Fatalf("foreign history caused API mutation: %v", action)
				}
			}
		})
	}
}

func TestUnlistedStatusRevisionLookup(t *testing.T) {
	for _, state := range []string{"owned", "orphan", "missing", "get-error", "listed"} {
		t.Run(state, func(t *testing.T) {
			set := newStatefulSet(1)
			revision := newRevisionOrDie(set, 1)
			set.Status.CurrentRevision = revision.Name
			set.Status.UpdateRevision = revision.Name
			if state != "listed" {
				revision.Labels = nil
			}
			if state == "orphan" {
				revision.OwnerReferences = nil
			}
			kube := fake.NewSimpleClientset()
			if state != "missing" && state != "get-error" {
				if err := kube.Tracker().Add(revision); err != nil {
					t.Fatal(err)
				}
			}
			lookupErr := fmt.Errorf("revision lookup unavailable")
			if state == "get-error" {
				kube.PrependReactor("get", "controllerrevisions", func(clienttesting.Action) (bool, runtime.Object, error) {
					return true, nil, lookupErr
				})
			}
			control := &defaultStatefulSetControl{csAppsV1: kube.AppsV1()}
			got, err := control.ListRevisions(set)
			if state == "get-error" {
				if err != lookupErr {
					t.Fatalf("lookup error was not propagated: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			} else if state == "missing" {
				if len(got) != 0 {
					t.Fatalf("missing revision returned: %v", got)
				}
			} else if len(got) != 1 || !reflect.DeepEqual(got[0], revision) {
				t.Fatalf("referenced history not preserved: %v", got)
			}
			gets := 0
			for _, action := range kube.Actions() {
				switch action.GetVerb() {
				case "get":
					gets++
				case "list":
				default:
					t.Fatalf("lookup caused API mutation: %v", action)
				}
			}
			wantGets := 1
			if state == "listed" {
				wantGets = 0
			}
			if gets != wantGets {
				t.Fatalf("got %d GETs, want %d", gets, wantGets)
			}
		})
	}
}
