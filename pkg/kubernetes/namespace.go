package kubernetes

import (
	"fmt"

	"github.com/rosskirkpat/kscaler/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a namespace.
func CreateNamespace(client client.Client, name string) error {
	err := client.Create(client.Ctx, name, &corev1.Namespace{}, &corev1.Namespace{}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", name, err)
	}
	return nil
}

// DeleteNamespace deletes a namespace.
func DeleteNamespace(client client.Client, name string) error {
	err := client.Delete(client.Ctx, name, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", name, err)
	}
	return nil
}
