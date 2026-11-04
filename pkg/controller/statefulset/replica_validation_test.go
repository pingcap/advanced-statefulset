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
	"math"
	"testing"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	pcfake "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestInvalidSlotsDoNotWriteObjects(t *testing.T) {
	for _, tc := range []struct {
		name, slots string
		replicas    int32
		nilReplicas bool
	}{
		{name: "negative-slot", slots: "[-1]", replicas: 1},
		{name: "malformed", slots: "[0,", replicas: 1},
		{name: "overflow", slots: "[0]", replicas: math.MaxInt32},
		{name: "huge", replicas: math.MaxInt32},
		{name: "negative-replicas", replicas: -1},
		{name: "nil-replicas", nilReplicas: true},
		{name: "ordinal-budget", slots: "[0]", replicas: helper.MaxReplicaCount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := newStatefulSet(int(tc.replicas))
			if tc.slots != "" {
				set.Annotations = map[string]string{helper.DeleteSlotsAnn: tc.slots}
			}
			if tc.nilReplicas {
				set.Spec.Replicas = nil
			}
			kube, pc := fake.NewSimpleClientset(), pcfake.NewSimpleClientset(set)
			_, _, control, stop := setupController(pc, kube)
			defer close(stop)
			kube.ClearActions()
			pc.ClearActions()
			if err := control.UpdateStatefulSet(set, nil); err == nil {
				t.Fatal("invalid replica configuration accepted")
			}
			if len(kube.Actions()) != 0 || len(pc.Actions()) != 0 {
				t.Fatal("invalid input caused API activity")
			}
		})
	}
}

func TestInvalidSlotsRejectedBeforeApplyingRevision(t *testing.T) {
	set := newStatefulSet(1)
	set.Annotations = map[string]string{helper.DeleteSlotsAnn: "[-1]"}
	revision := newRevisionOrDie(set, 1)
	_, err := (&defaultStatefulSetControl{}).updateStatefulSet(set, revision, revision, 0, nil)
	if err == nil {
		t.Fatal("invalid replica configuration accepted")
	}
}
