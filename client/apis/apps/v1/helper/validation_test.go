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
	"strings"
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
		{"zero-slot", "[0]", 1, true, []int32{1}},
		{"negative", "[-1]", 1, false, nil},
		{"malformed", "[0,", 1, false, nil},
		{"null", "null", 1, true, []int32{0}},
		{"null-element", "[null]", 1, true, []int32{1}},
		{"mixed-null-element", "[1,null]", 1, true, []int32{2}},
		{"out-of-int32", "[2147483648]", 1, false, nil},
		{"overflow", "[0]", math.MaxInt32, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := &metav1.ObjectMeta{Annotations: map[string]string{DeleteSlotsAnn: tc.slots}}
			got := sets.NewInt32()
			err := VisitPodOrdinalsChecked(tc.replicas, obj, func(ordinal int32) bool {
				got.Insert(ordinal)
				return true
			})
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
	for _, slots := range []string{"[bad", "[-1]", "[2147483648]"} {
		t.Run("AddDeleteSlots/"+slots, func(t *testing.T) {
			obj := &metav1.ObjectMeta{Annotations: map[string]string{DeleteSlotsAnn: slots, "other": "keep"}}
			if err := AddDeleteSlots(obj, sets.NewInt32(0)); err == nil {
				t.Error("AddDeleteSlots accepted invalid metadata")
			}
			if obj.Annotations[DeleteSlotsAnn] != slots || obj.Annotations["other"] != "keep" {
				t.Fatal("invalid metadata was changed")
			}
		})
	}
}

// TiDB Operator v1 uses JSON []int32 parsing and the legacy helpers. Preserve
// its non-negative input semantics, including nulls, duplicates and ignored slots.
func TestTiDBOperatorDeleteSlotsCompatibility(t *testing.T) {
	for _, slots := range []string{"null", "[null]", "[1,null]", "[]", "[0,2,2,2147483647]", "[" + strings.Repeat("0,", 10000) + "0]"} {
		obj := &metav1.ObjectMeta{Annotations: map[string]string{DeleteSlotsAnn: slots}}
		got := sets.NewInt32()
		err := VisitPodOrdinalsChecked(3, obj, func(ordinal int32) bool {
			got.Insert(ordinal)
			return true
		})
		if err != nil || !got.Equal(GetPodOrdinals(3, obj)) {
			t.Fatalf("slots %q: got %v, error %v", slots, got, err)
		}
		if err := AddDeleteSlots(obj, sets.NewInt32(1)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVisitHugeOrdinalRange(t *testing.T) {
	obj := &metav1.ObjectMeta{}
	visited := 0
	if err := VisitPodOrdinalsChecked(math.MaxInt32, obj, func(ordinal int32) bool {
		if ordinal != int32(visited) {
			t.Fatalf("got ordinal %d, want %d", ordinal, visited)
		}
		visited++
		return visited < 3
	}); err != nil || visited != 3 {
		t.Fatalf("visited %d, error %v", visited, err)
	}
	// Overflow must fail before invoking the visitor, without wrapping or clamping.
	obj.Annotations = map[string]string{DeleteSlotsAnn: "[0]"}
	if err := VisitPodOrdinalsChecked(math.MaxInt32, obj, func(int32) bool {
		t.Fatal("visitor called for an overflowing range")
		return false
	}); err == nil {
		t.Fatal("overflow accepted")
	}
}
