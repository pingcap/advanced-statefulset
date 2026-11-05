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
	"fmt"
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	DeleteSlotsAnn = "delete-slots"

	// PausedReconcileAnn is the annotation key for the paused reconcile.
	// If the value is "true", the controller will not reconcile the statefulset.
	// The controller will not delete the pods, update the status, etc.
	// We use an annotation instead of a field in the status
	// so that we can convert between the K8s built-in StatefulSet and ours.
	PausedReconcileAnn = "paused-reconcile"
)

// Deprecated: use GetDeleteSlotsChecked when reading untrusted annotations.
func GetDeleteSlots(set metav1.Object) (deleteSlots sets.Int32) {
	deleteSlots = sets.NewInt32()
	annotations := set.GetAnnotations()
	if annotations == nil {
		return
	}
	value, ok := annotations[DeleteSlotsAnn]
	if !ok {
		return
	}
	var slice []int32
	err := json.Unmarshal([]byte(value), &slice)
	if err != nil {
		return
	}
	deleteSlots.Insert(slice...)
	return
}

func SetDeleteSlots(set metav1.Object, deleteSlots sets.Int32) (err error) {
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

func AddDeleteSlots(set metav1.Object, deleteSlots sets.Int32) (err error) {
	currentDeleteSlots, err := GetDeleteSlotsChecked(set)
	if err != nil {
		return err
	}
	return SetDeleteSlots(set, currentDeleteSlots.Union(deleteSlots))
}

// GetMaxReplicaCountAndDeleteSlots returns the max replica count and delete
// slots. The desired slots of this stateful set will be [0, replicaCount) - [delete slots].
// Deprecated: use GetMaxReplicaCountAndDeleteSlotsChecked for bounded reconciliation.
func GetMaxReplicaCountAndDeleteSlots(replicas int32, deleteSlots sets.Int32) (int32, sets.Int32) {
	replicaCount := replicas
	deleteSlotsCopy := sets.NewInt32()
	for k := range deleteSlots {
		deleteSlotsCopy.Insert(k)
	}
	for _, deleteSlot := range deleteSlotsCopy.List() {
		if deleteSlot < replicaCount {
			replicaCount++
		} else {
			deleteSlotsCopy.Delete(deleteSlot)
		}
	}
	return replicaCount, deleteSlotsCopy
}

func SetPausedReconcile(set metav1.Object, paused bool) {
	annotations := set.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	if paused {
		annotations[PausedReconcileAnn] = "true"
	} else {
		delete(annotations, PausedReconcileAnn)
	}
	set.SetAnnotations(annotations)
}

func GetPausedReconcile(set metav1.Object) bool {
	annotations := set.GetAnnotations()
	if annotations == nil {
		return false
	}
	value, ok := annotations[PausedReconcileAnn]
	if !ok {
		return false
	}
	return value == "true"
}

func GetPodOrdinals(replicas int32, set metav1.Object) sets.Int32 {
	return GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, GetDeleteSlots(set))
}

func GetPodOrdinalsFromReplicasAndDeleteSlots(replicas int32, deleteSlots sets.Int32) sets.Int32 {
	maxReplicaCount, deleteSlots := GetMaxReplicaCountAndDeleteSlots(replicas, deleteSlots)
	podOrdinals := sets.NewInt32()
	for i := int32(0); i < maxReplicaCount; i++ {
		if !deleteSlots.Has(i) {
			podOrdinals.Insert(i)
		}
	}
	return podOrdinals
}

func GetMaxPodOrdinal(replicas int32, set metav1.Object) int32 {
	var max int32
	max = -1
	for k := range GetPodOrdinals(replicas, set) {
		if k > max {
			max = k
		}
	}
	return max
}

func GetMinPodOrdinal(replicas int32, set metav1.Object) int32 {
	var min int32
	min = math.MaxInt32
	for k := range GetPodOrdinals(replicas, set) {
		if k < min {
			min = k
		}
	}
	return min
}

// MaxReplicaCount bounds the ordinal range materialized by the controller.
const MaxReplicaCount int32 = 10000

// GetDeleteSlotsChecked rejects malformed, negative and excessive delete slots.
func GetDeleteSlotsChecked(set metav1.Object) (sets.Int32, error) {
	value, exists := set.GetAnnotations()[DeleteSlotsAnn]
	if !exists {
		return sets.NewInt32(), nil
	}
	var slots []*int32
	if err := json.Unmarshal([]byte(value), &slots); err != nil || slots == nil {
		return nil, fmt.Errorf("delete-slots must be a JSON array of non-negative int32 ordinals")
	}
	if len(slots) > int(MaxReplicaCount) {
		return nil, fmt.Errorf("too many delete-slots (maximum %d)", MaxReplicaCount)
	}
	deleteSlots := sets.NewInt32()
	for _, slot := range slots {
		if slot == nil {
			return nil, fmt.Errorf("delete-slots must not contain null ordinals")
		}
		if *slot < 0 {
			return nil, fmt.Errorf("delete-slots must not contain negative ordinals")
		}
		deleteSlots.Insert(*slot)
	}
	return deleteSlots, nil
}

// GetMaxReplicaCountAndDeleteSlotsChecked validates before any dense allocation.
func GetMaxReplicaCountAndDeleteSlotsChecked(replicas int32, deleteSlots sets.Int32) (int32, sets.Int32, error) {
	if replicas < 0 || replicas > MaxReplicaCount {
		return 0, nil, fmt.Errorf("replicas must be between 0 and %d", MaxReplicaCount)
	}
	if deleteSlots.Len() > int(MaxReplicaCount) {
		return 0, nil, fmt.Errorf("too many delete-slots")
	}
	count := int64(replicas)
	effective := sets.NewInt32()
	for _, slot := range deleteSlots.List() {
		if slot < 0 {
			return 0, nil, fmt.Errorf("delete-slots must not contain negative ordinals")
		}
		if int64(slot) < count {
			count++
			if count > int64(MaxReplicaCount) {
				return 0, nil, fmt.Errorf("replica ordinal range exceeds maximum %d", MaxReplicaCount)
			}
			effective.Insert(slot)
		}
	}
	return int32(count), effective, nil
}

// GetPodOrdinalsChecked is the error-returning counterpart of GetPodOrdinals.
func GetPodOrdinalsChecked(replicas int32, set metav1.Object) (sets.Int32, error) {
	slots, err := GetDeleteSlotsChecked(set)
	if err != nil {
		return nil, err
	}
	count, slots, err := GetMaxReplicaCountAndDeleteSlotsChecked(replicas, slots)
	if err != nil {
		return nil, err
	}
	ordinals := sets.NewInt32()
	for i := int32(0); i < count; i++ {
		if !slots.Has(i) {
			ordinals.Insert(i)
		}
	}
	return ordinals, nil
}
