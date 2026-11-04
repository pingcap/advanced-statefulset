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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	kubeapps "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMigrationRevisionRequiresSourceUID(t *testing.T) {
	set := newStatefulSet(1)
	set.UID = "asts-uid"
	set.Annotations = map[string]string{"apps.pingcap.com/upgrade-from-statefulset-uid": "expected-source"}
	revision := newRevisionOrDie(set, 1)
	revision.OwnerReferences[0].APIVersion = kubeapps.SchemeGroupVersion.String()
	revision.OwnerReferences[0].UID = "different-source"
	markRevisionForUpgrade(set, revision)
	control := &defaultStatefulSetControl{csAppsV1: fake.NewSimpleClientset(revision).AppsV1()}
	got, err := control.ListRevisions(set)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatal("same-name revision from a different source UID was accepted")
	}
}

func TestLegacyMigrationWaitsForOrphaning(t *testing.T) {
	set := newStatefulSet(1)
	set.UID = "asts-uid"
	revision := newRevisionOrDie(set, 1)
	revision.OwnerReferences[0].APIVersion = kubeapps.SchemeGroupVersion.String()
	revision.OwnerReferences[0].UID = "legacy-source"
	markRevisionForUpgrade(set, revision)
	set.Annotations = nil
	revision.Annotations = nil
	set.Status.CurrentRevision = revision.Name
	set.Status.UpdateRevision = revision.Name
	kube := fake.NewSimpleClientset(revision)
	control := &defaultStatefulSetControl{csAppsV1: kube.AppsV1()}
	if _, err := control.ListRevisions(set); err == nil {
		t.Fatal("unbound builtin history should wait for GC")
	}
	revision.OwnerReferences = nil
	if _, err := kube.AppsV1().ControllerRevisions(set.Namespace).Update(context.Background(), revision, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	got, err := control.ListRevisions(set)
	if err != nil || len(got) != 1 {
		t.Fatalf("legacy referenced orphan lost: %v %v", got, err)
	}
}

func TestForgedOrphanMarkerAndEmptySelector(t *testing.T) {
	set := newStatefulSet(1)
	set.UID = "asts-uid"
	revision := newRevisionOrDie(set, 1)
	revision.OwnerReferences = nil
	markRevisionForUpgrade(set, revision)
	revision.Annotations = nil
	kube := fake.NewSimpleClientset(revision)
	control := &defaultStatefulSetControl{csAppsV1: kube.AppsV1()}
	got, err := control.ListRevisions(set)
	if err != nil || len(got) != 0 {
		t.Fatalf("unbound marker accepted: %v %v", got, err)
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
