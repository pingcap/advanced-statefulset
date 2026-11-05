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

package helper

import (
	"context"
	"testing"

	asv1 "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	asfake "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	kt "k8s.io/client-go/testing"
)

func TestUpgradeOnlyMarksSourceOwnedHistory(t *testing.T) {
	ctx := context.Background()
	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns", UID: "source-uid"}, Spec: appsv1.StatefulSetSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "db"}}}}
	revision := func(name string, owner *metav1.OwnerReference) *appsv1.ControllerRevision {
		r := &appsv1.ControllerRevision{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns", Labels: map[string]string{"app": "db"}}}
		if owner != nil {
			r.OwnerReferences = []metav1.OwnerReference{*owner}
		}
		return r
	}
	owner := metav1.NewControllerRef(sts, appsv1.SchemeGroupVersion.WithKind("StatefulSet"))
	foreign := *owner
	foreign.UID = "foreign-uid"
	kube := fake.NewSimpleClientset(sts, revision("owned", owner), revision("foreign", &foreign), revision("orphan", nil))
	target := &asv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns", UID: "new-asts-uid"}}
	asc := asfake.NewSimpleClientset(target)
	deleted := false
	kube.PrependReactor("delete", "statefulsets", func(a kt.Action) (bool, runtime.Object, error) {
		opts := a.(kt.DeleteAction).GetDeleteOptions()
		if opts.Preconditions == nil || opts.Preconditions.UID == nil || *opts.Preconditions.UID != sts.UID {
			t.Fatal("source deletion has no UID precondition")
		}
		deleted = true
		return false, nil, nil
	})
	got, err := Upgrade(ctx, kube, asc, sts)
	if err != nil {
		t.Fatal(err)
	}
	if got.UID != types.UID("new-asts-uid") || got.Annotations[UpgradeFromStatefulSetUIDAnn] != string(sts.UID) {
		t.Fatal("migration source binding missing or new identity changed")
	}
	if !deleted {
		t.Fatal("source not deleted")
	}
	for _, name := range []string{"owned", "foreign", "orphan"} {
		r, err := kube.AppsV1().ControllerRevisions("ns").Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if name == "owned" {
			if r.Labels[UpgradeToAdvancedStatefulSetAnn] != "db" || r.Annotations[UpgradeFromStatefulSetUIDAnn] != string(sts.UID) {
				t.Fatal("owned revision not bound")
			}
		} else if r.Labels["app"] != "db" || r.Labels[UpgradeToAdvancedStatefulSetAnn] != "" {
			t.Fatal("foreign/orphan history was rewritten", name)
		}
	}
	if _, err := Upgrade(ctx, kube, asc, sts); err != nil {
		t.Fatal("completed migration retry failed", err)
	}
}

func TestUpgradeRejectsRecreatedSourceBeforeWriting(t *testing.T) {
	old := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns", UID: "old"}}
	fresh := old.DeepCopy()
	fresh.UID = "replacement"
	kube, asc := fake.NewSimpleClientset(fresh), asfake.NewSimpleClientset()
	if _, err := Upgrade(context.Background(), kube, asc, old); err == nil {
		t.Fatal("stale source accepted")
	}
	for _, a := range kube.Actions() {
		if a.GetVerb() != "get" {
			t.Fatal("stale source caused write", a)
		}
	}
	if len(asc.Actions()) != 0 {
		t.Fatal("stale source touched target")
	}
}

func TestUpgradeRetryBindsPreviouslyMarkedHistory(t *testing.T) {
	ctx := context.Background()
	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns", UID: "source"}, Spec: appsv1.StatefulSetSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "db"}}}}
	revision := &appsv1.ControllerRevision{ObjectMeta: metav1.ObjectMeta{Name: "old-current", Namespace: "ns", Labels: map[string]string{UpgradeToAdvancedStatefulSetAnn: "db"}, OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(sts, appsv1.SchemeGroupVersion.WithKind("StatefulSet"))}}}
	sts.Status.CurrentRevision = revision.Name
	sts.Status.UpdateRevision = revision.Name
	kube, asc := fake.NewSimpleClientset(sts, revision), asfake.NewSimpleClientset()
	got, err := Upgrade(ctx, kube, asc, sts)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := kube.AppsV1().ControllerRevisions("ns").Get(ctx, revision.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if bound.Annotations[UpgradeFromStatefulSetUIDAnn] != string(sts.UID) || got.Status.CurrentRevision != revision.Name {
		t.Fatal("legacy retry history was not bound")
	}
}
