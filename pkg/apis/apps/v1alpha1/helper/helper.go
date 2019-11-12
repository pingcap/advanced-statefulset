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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	deletedSlotsAnn = "deleted-slots"
)

func GetDeletedSlots(set metav1.Object) (deletedSlots sets.Int) {
	deletedSlots = sets.NewInt()
	annotations := set.GetAnnotations()
	if annotations == nil {
		return
	}
	value, ok := annotations[deletedSlotsAnn]
	if !ok {
		return
	}
	var slice []int
	err := json.Unmarshal([]byte(value), &slice)
	if err != nil {
		return
	}
	deletedSlots.Insert(slice...)
	return
}

func SetDeletedSlots(set metav1.Object, deletedSlots sets.Int) (err error) {
	annotations := set.GetAnnotations()
	if deletedSlots == nil || deletedSlots.Len() == 0 {
		// clear
		delete(annotations, deletedSlotsAnn)
	} else {
		// set
		b, err := json.Marshal(deletedSlots.List())
		if err != nil {
			return err
		}
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[deletedSlotsAnn] = string(b)
	}
	set.SetAnnotations(annotations)
	return
}

func AddDeletedSlots(set metav1.Object, deletedSlots sets.Int) (err error) {
	currentDeletedSlots := GetDeletedSlots(set)
	return SetDeletedSlots(set, currentDeletedSlots.Union(deletedSlots))
}

// GetMaxReplicaCountAndDeletedSlots returns the max replica count and delete
// slots. The desired slots of this stateful set will be [0, replicaCount) - [delete slots].
func GetMaxReplicaCountAndDeletedSlots(replicas int, deletedSlots sets.Int) (int, sets.Int) {
	replicaCount := replicas
	for _, deleteSlot := range deletedSlots.List() {
		if deleteSlot < replicaCount {
			replicaCount++
		} else {
			deletedSlots.Delete(deleteSlot)
		}
	}
	return replicaCount, deletedSlots
}
