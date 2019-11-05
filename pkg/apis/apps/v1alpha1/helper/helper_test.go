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

package helper

import (
	"testing"

	apps "github.com/cofyc/advanced-statefulset/pkg/apis/apps/v1alpha1"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGetDeleteSlots(t *testing.T) {
	tests := []struct {
		name string
		sts  apps.StatefulSet
		want sets.Int
	}{
		{
			name: "no annotation",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
			},
			want: nil,
		},
		{
			name: "empty annotation",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "",
					},
				},
			},
			want: nil,
		},
		{
			name: "invalid annotation",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "invalid",
					},
				},
			},
			want: nil,
		},
		{
			name: "vailid annotation with one value",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "[1]",
					},
				},
			},
			want: sets.NewInt(1),
		},
		{
			name: "vailid annotation with multiple values",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "[1, 2, 3]",
					},
				},
			},
			want: sets.NewInt(1, 2, 3),
		},
		{
			name: "vailid annotation with duplicate values",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "[1, 2, 3, 3]",
					},
				},
			},
			want: sets.NewInt(1, 2, 3),
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
		sts  apps.StatefulSet
		set  sets.Int
		want apps.StatefulSet
	}{
		{
			name: "nil int set",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "[1]",
					},
				},
			},
			set: nil,
			want: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
		{
			name: "empty int set",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "[1]",
					},
				},
			},
			set: sets.NewInt(),
			want: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
		{
			name: "one int set",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
			},
			set: sets.NewInt(3),
			want: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "[3]",
					},
				},
			},
		},
		{
			name: "multiple ints set",
			sts: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{},
			},
			set: sets.NewInt(3, 4, 1),
			want: apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						deletedSlotsAnnotation: "[1,3,4]",
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
