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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	DeleteSlotsAnn = "delete-slots"
)

func GetDeleteSlots(set metav1.Object) (deleteSlots sets.Int) {
	deleteSlots = sets.NewInt()
	annotations := set.GetAnnotations()
	if annotations == nil {
		return
	}
	value, ok := annotations[DeleteSlotsAnn]
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

func SetDeleteSlots(set metav1.Object, deleteSlots sets.Int) (err error) {
	annotations := set.GetAnnotations()
	if deleteSlots == nil || deleteSlots.Len() == 0 {
		// clear
		delete(annotations, DeleteSlotsAnn)
	} else {
		// set
		b, err := json.Marshal(deleteSlots.List())
		if err != nil {
			return err
		}
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[DeleteSlotsAnn] = string(b)
	}
	set.SetAnnotations(annotations)
	return
}

func AddDeleteSlots(set metav1.Object, deleteSlots sets.Int) (err error) {
	currentDeleteSlots := GetDeleteSlots(set)
	return SetDeleteSlots(set, currentDeleteSlots.Union(deleteSlots))
}

// GetMaxReplicaCountAndDeleteSlots returns the max replica count and delete
// slots. The desired slots of this stateful set will be [0, replicaCount) - [delete slots].
func GetMaxReplicaCountAndDeleteSlots(replicas int, deleteSlots sets.Int) (int, sets.Int) {
	replicaCount := replicas
	for _, deleteSlot := range deleteSlots.List() {
		if deleteSlot < replicaCount {
			replicaCount++
		} else {
			deleteSlots.Delete(deleteSlot)
		}
	}
	return replicaCount, deleteSlots
}

// GetDesiredPodOrdinals gets desired pod ordinals of given statefulset set.
func GetDesiredPodOrdinals(replicas int, set metav1.Object) sets.Int {
	maxReplicaCount, deleteSlots := GetMaxReplicaCountAndDeleteSlots(replicas, GetDeleteSlots(set))
	podOrdinals := sets.NewInt()
	for i := 0; i < maxReplicaCount; i++ {
		if !deleteSlots.Has(i) {
			podOrdinals.Insert(i)
		}
	}
	return podOrdinals
}
