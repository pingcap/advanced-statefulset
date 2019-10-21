package statefulset

import (
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getCustomResourceDefinitionData() []*apiextensionsv1beta1.CustomResourceDefinition {
	return []*apiextensionsv1beta1.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "statefulsets.pingcap.com",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   "pingcap.com",
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controllerrevisions.pingcap.com",
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   "pingcap.com",
				Version: "v1alpha1",
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Plural:   "controllerrevisions",
					Singular: "controllerrevision",
					Kind:     "ControllerRevision",
				},
			},
		},
	}
}
