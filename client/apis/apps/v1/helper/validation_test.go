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
	"math"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestCheckedOrdinals(t *testing.T) {
	for _, tc := range []struct {
		name, slots string
		replicas    int32
		valid       bool
		want        []int32
	}{
		{"normal", "[0,2,2,2147483647]", 3, true, []int32{1, 3, 4}},
		{"zero", "[0]", 0, true, []int32{}},
		{"negative", "[-1]", 1, false, nil},
		{"malformed", "[0,", 1, false, nil},
		{"null", "null", 1, false, nil},
		{"out-of-int32", "[2147483648]", 1, false, nil},
		{"overflow", "[0]", math.MaxInt32, false, nil},
		{"huge", "[]", math.MaxInt32, false, nil},
		{"budget", "[0]", MaxReplicaCount, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := &metav1.ObjectMeta{Annotations: map[string]string{DeleteSlotsAnn: tc.slots}}
			got, err := GetPodOrdinalsChecked(tc.replicas, obj)
			if (err == nil) != tc.valid {
				t.Fatalf("got %v, error %v", got, err)
			}
			if tc.valid && !got.Equal(sets.NewInt32(tc.want...)) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	if _, _, err := GetMaxReplicaCountAndDeleteSlotsChecked(1, sets.NewInt32(-1)); err == nil {
		t.Fatal("negative set accepted")
	}
	obj := &metav1.ObjectMeta{Annotations: map[string]string{DeleteSlotsAnn: "[bad"}}
	if err := AddDeleteSlots(obj, sets.NewInt32(0)); err == nil {
		t.Fatal("AddDeleteSlots overwrote malformed metadata")
	}
	if obj.Annotations[DeleteSlotsAnn] != "[bad" {
		t.Fatal("malformed slots were changed")
	}
}
