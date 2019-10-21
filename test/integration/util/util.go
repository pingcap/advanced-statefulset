package util

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateTestingNamespace creates a namespace for testing.
func CreateTestingNamespace(baseName string, client kubernetes.Interface, t *testing.T) *v1.Namespace {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: baseName,
		},
	}
	ns, err := client.CoreV1().Namespaces().Create(ns)
	if err != nil {
		t.Fatal(err)
	}
	return ns
}

// DeleteTestingNamespace is currently a no-op function.
func DeleteTestingNamespace(ns *v1.Namespace, client kubernetes.Interface, t *testing.T) {
	// We are using different namespace for each, so no conflicts for now.
}
