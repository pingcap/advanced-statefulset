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
	"testing"

	"github.com/google/go-cmp/cmp"
	asappsv1 "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGetDeleteSlots(t *testing.T) {
	tests := []struct {
		name string
		sts  asappsv1.StatefulSet
		want sets.Int32
	}{
		{
			name: "no annotation",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
			},
			want: nil,
		},
		{
			name: "empty annotation",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "",
					},
				},
			},
			want: nil,
		},
		{
			name: "invalid annotation",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "invalid",
					},
				},
			},
			want: nil,
		},
		{
			name: "vailid annotation with one value",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[1]",
					},
				},
			},
			want: sets.NewInt32(1),
		},
		{
			name: "vailid annotation with multiple values",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[1, 2, 3]",
					},
				},
			},
			want: sets.NewInt32(1, 2, 3),
		},
		{
			name: "vailid annotation with duplicate values",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[1, 2, 3, 3]",
					},
				},
			},
			want: sets.NewInt32(1, 2, 3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDeleteSlots(&tt.sts)
			if !got.Equal(tt.want) {
				t.Errorf("GetDeleteSlots want %v got %v", tt.want, got)
			}
		})
	}
}

func TestSetDeleteSlots(t *testing.T) {
	tests := []struct {
		name string
		sts  asappsv1.StatefulSet
		set  sets.Int32
		want asappsv1.StatefulSet
	}{
		{
			name: "nil int set",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[1]",
					},
				},
			},
			set: nil,
			want: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
		{
			name: "empty int set",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[1]",
					},
				},
			},
			set: sets.NewInt32(),
			want: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
		{
			name: "one int set",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
			},
			set: sets.NewInt32(3),
			want: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[3]",
					},
				},
			},
		},
		{
			name: "multiple ints set",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
			},
			set: sets.NewInt32(3, 4, 1),
			want: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[1,3,4]",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = SetDeleteSlots(&tt.sts, tt.set)
			if diff := cmp.Diff(tt.want, tt.sts); diff != "" {
				t.Errorf("unexpected result (-want, +got): %s", diff)
			}
		})
	}
}

func TestSetPausedReconcile(t *testing.T) {
	tests := []struct {
		name string
		set  bool
		sts  asappsv1.StatefulSet
		want asappsv1.StatefulSet
	}{
		{
			name: "set paused reconcile",
			set:  true,
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
			},
			want: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						PausedReconcileAnn: "true",
					},
				},
			},
		},
		{
			name: "unset paused reconcile",
			set:  false,
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						PausedReconcileAnn: "true",
					},
				},
			},
			want: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
		{
			name: "toggle paused reconcile to false",
			set:  false,
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						PausedReconcileAnn: "true",
					},
				},
			},
			want: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetPausedReconcile(&tt.sts, tt.set)
			if diff := cmp.Diff(tt.want, tt.sts); diff != "" {
				t.Errorf("unexpected result (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetPausedReconcile(t *testing.T) {
	tests := []struct {
		name string
		sts  asappsv1.StatefulSet
		want bool
	}{
		{
			name: "no annotation",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
			},
			want: false,
		},
		{
			name: "empty annotation",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			want: false,
		},
		{
			name: "paused reconcile",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						PausedReconcileAnn: "true",
					},
				},
			},
			want: true,
		},
		{
			name: "not paused reconcile",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						PausedReconcileAnn: "false",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPausedReconcile(&tt.sts)
			if got != tt.want {
				t.Errorf("GetPausedReconcile want %v got %v", tt.want, got)
			}
		})
	}
}

func int32ptr(i int32) *int32 {
	return &i
}

func TestGetPodOrdinals(t *testing.T) {
	tests := []struct {
		name string
		sts  asappsv1.StatefulSet
		want sets.Int32
	}{
		{
			name: "no delete slots",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: asappsv1.StatefulSetSpec{
					Replicas: int32ptr(3),
				},
			},
			want: sets.NewInt32(0, 1, 2),
		},
		{
			name: "delete slots in [0, replicas)",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[0, 2]",
					},
				},
				Spec: asappsv1.StatefulSetSpec{
					Replicas: int32ptr(3),
				},
			},
			want: sets.NewInt32(1, 3, 4),
		},
		{
			name: "delete slots not in [0, replicas)",
			sts: asappsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteSlotsAnn: "[4, 5]",
					},
				},
				Spec: asappsv1.StatefulSetSpec{
					Replicas: int32ptr(3),
				},
			},
			want: sets.NewInt32(0, 1, 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPodOrdinals(*tt.sts.Spec.Replicas, &tt.sts)
			if diff := cmp.Diff(tt.want.List(), got.List()); diff != "" {
				t.Errorf("unexpected result (-want, +got): %s", diff)
			}
		})
	}
}
