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
	"encoding/json"

	apps "github.com/cofyc/advanced-statefulset/pkg/apis/apps/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	deletedSlotsAnnotation = "delete-slots"
)

func GetDeleteSlots(set *apps.StatefulSet) (deleteSlots sets.Int) {
	deleteSlots = sets.NewInt()
	if set.Annotations == nil {
		return
	}
	value, ok := set.Annotations[deletedSlotsAnnotation]
	if !ok {
		return
	}
	var slice []int
	err := json.Unmarshal([]byte(value), &slice)
	if err != nil {
		return
	}
	deleteSlots.Insert(slice...)
	return
}

func SetDeleteSlots(set *apps.StatefulSet, deleteSlots sets.Int) (err error) {
	if deleteSlots == nil || deleteSlots.Len() == 0 {
		// clear the annotation
		if set.ObjectMeta.Annotations != nil {
			delete(set.ObjectMeta.Annotations, deletedSlotsAnnotation)
		}
		return
	}
	b, err := json.Marshal(deleteSlots.List())
	if err != nil {
		return
	}
	metav1.SetMetaDataAnnotation(&set.ObjectMeta, deletedSlotsAnnotation, string(b))
	return
}

func AddDeleteSlots(set *apps.StatefulSet, deleteSlots sets.Int) (err error) {
	currentDeleteSlots := GetDeleteSlots(set)
	return SetDeleteSlots(set, currentDeleteSlots.Union(deleteSlots))
}
