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
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// read from deployment/crd.v1.yaml
func getCustomResourceDefinitionData() []*apiextensionsv1beta1.CustomResourceDefinition {
	return []*apiextensionsv1beta1.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "statefulsets.apps.pingcap.com",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   "apps.pingcap.com",
				Version: "v1alpha1",
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Plural:   "statefulsets",
					Singular: "statefulset",
					Kind:     "StatefulSet",
					ShortNames: []string{
						"sts",
					},
				},
				Versions: []apiextensionsv1beta1.CustomResourceDefinitionVersion{
					{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
					},
				},
				Subresources: &apiextensionsv1beta1.CustomResourceSubresources{
					Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
					Scale: &apiextensionsv1beta1.CustomResourceSubresourceScale{
						SpecReplicasPath:   ".spec.replicas",
						StatusReplicasPath: ".status.replicas",
						LabelSelectorPath: func(a string) *string {
							return &a
						}(".status.labelSelector"),
					},
				},
			},
		},
	}
}
