/*
Copyright 2015 The Kubernetes Authors.

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

// this file is copied from k8s.io/kubernetes/test/e2e/framework/pv/pv.go @v1.23.17

package pv

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/pingcap/advanced-statefulset/test/third_party/k8s"
	e2eskipper "github.com/pingcap/advanced-statefulset/test/third_party/k8s/skipper"
)

const (
	// isDefaultStorageClassAnnotation represents a StorageClass annotation that
	// marks a class as the default StorageClass
	isDefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

	// betaIsDefaultStorageClassAnnotation is the beta version of IsDefaultStorageClassAnnotation.
	// TODO: remove Beta when no longer used
	betaIsDefaultStorageClassAnnotation = "storageclass.beta.kubernetes.io/is-default-class"
)

// GetDefaultStorageClassName returns default storageClass or return error
func GetDefaultStorageClassName(c clientset.Interface) (string, error) {
	list, err := c.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("Error listing storage classes: %v", err)
	}
	var scName string
	for _, sc := range list.Items {
		if isDefaultAnnotation(sc.ObjectMeta) {
			if len(scName) != 0 {
				return "", fmt.Errorf("Multiple default storage classes found: %q and %q", scName, sc.Name)
			}
			scName = sc.Name
		}
	}
	if len(scName) == 0 {
		return "", fmt.Errorf("No default storage class found")
	}
	k8s.Logf("Default storage class: %q", scName)
	return scName, nil
}

// isDefaultAnnotation returns a boolean if the default storage class
// annotation is set
// TODO: remove Beta when no longer needed
func isDefaultAnnotation(obj metav1.ObjectMeta) bool {
	if obj.Annotations[isDefaultStorageClassAnnotation] == "true" {
		return true
	}
	if obj.Annotations[betaIsDefaultStorageClassAnnotation] == "true" {
		return true
	}

	return false
}

// SkipIfNoDefaultStorageClass skips tests if no default SC can be found.
func SkipIfNoDefaultStorageClass(c clientset.Interface) {
	_, err := GetDefaultStorageClassName(c)
	if err != nil {
		e2eskipper.Skipf("error finding default storageClass : %v", err)
	}
}
