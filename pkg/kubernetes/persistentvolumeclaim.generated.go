package kubernetes

// Code generated by stub-gen; DO NOT EDIT.

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rosskirkpat/kscaler/pkg/client"
)

// PersistentVolumeClaim wraps a Kubernetes PersistentVolumeClaim.
type PersistentVolumeClaim struct {
	corev1.PersistentVolumeClaim

	client client.Client
}

// NewPersistentVolumeClaim creates a PersistentVolumeClaim from its Kubernetes PersistentVolumeClaim.
func NewPersistentVolumeClaim(client client.Client, persistentvolumeclaim corev1.PersistentVolumeClaim) (PersistentVolumeClaim, error) {
	createdPersistentVolumeClaim, err := client.Kubernetes.
		CoreV1().
		PersistentVolumeClaims(persistentvolumeclaim.Namespace).
		Create(client.Ctx, &persistentvolumeclaim, metav1.CreateOptions{})
	if err != nil {
		return PersistentVolumeClaim{}, fmt.Errorf("failed to create persistentvolumeclaim %s in namespace %s: %w", persistentvolumeclaim.Name, persistentvolumeclaim.Namespace, err)
	}

	return PersistentVolumeClaim{
		PersistentVolumeClaim: *createdPersistentVolumeClaim,
		client: client,
	}, nil
}

// GetPersistentVolumeClaim gets a persistentvolumeclaim in a namespace.
func GetPersistentVolumeClaim(client client.Client, name string, namespace string) (PersistentVolumeClaim, error) {
	options := metav1.GetOptions{}

	persistentvolumeclaim, err := client.Kubernetes.
		CoreV1().
		PersistentVolumeClaims(namespace).
		Get(client.Ctx, name, options)
	if err != nil {
		return PersistentVolumeClaim{}, fmt.Errorf("failed to get persistentvolumeclaim %s in namespace %s: %w", name, namespace, err)
	}

	return PersistentVolumeClaim{
		PersistentVolumeClaim: *persistentvolumeclaim,
		client: client,
	}, nil
}

// ListPersistentVolumeClaims lists all persistentvolumeclaims in a namespace.
func ListPersistentVolumeClaims(client client.Client, namespace string) ([]PersistentVolumeClaim, error) {
	options := metav1.ListOptions{}

	list, err := client.Kubernetes.
		CoreV1().
		PersistentVolumeClaims(namespace).
		List(client.Ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list persistentvolumeclaims in namespace %s: %w", namespace, err)
	}

	persistentvolumeclaims := make([]PersistentVolumeClaim, 0, len(list.Items))

	for _, item := range list.Items {
		persistentvolumeclaims = append(persistentvolumeclaims, PersistentVolumeClaim{
			PersistentVolumeClaim: item,
			client: client,
		})
	}

	return persistentvolumeclaims, nil
}

// Delete deletes a PersistentVolumeClaim from the Kubernetes cluster.
func (persistentvolumeclaim PersistentVolumeClaim) Delete() error {
	options := metav1.DeleteOptions{}

	err := persistentvolumeclaim.client.Kubernetes.
		CoreV1().
		PersistentVolumeClaims(persistentvolumeclaim.Namespace).
		Delete(persistentvolumeclaim.client.Ctx, persistentvolumeclaim.Name, options)
	if err != nil {
		return fmt.Errorf("failed to delete persistentvolumeclaim %s in namespace %s: %w", persistentvolumeclaim.Name, persistentvolumeclaim.Namespace, err)
	}

	return nil
}

// Update gets the current PersistentVolumeClaim status.
func (persistentvolumeclaim *PersistentVolumeClaim) Update() error {
	options := metav1.GetOptions{}

	update, err := persistentvolumeclaim.client.Kubernetes.
		CoreV1().
		PersistentVolumeClaims(persistentvolumeclaim.Namespace).
		Get(persistentvolumeclaim.client.Ctx, persistentvolumeclaim.Name, options)
	if err != nil {
		return fmt.Errorf("failed to update persistentvolumeclaim %s in namespace %s: %w", persistentvolumeclaim.Name, persistentvolumeclaim.Namespace, err)
	}

	persistentvolumeclaim.PersistentVolumeClaim = *update

	return nil
}

// Save saves the current PersistentVolumeClaim.
func (persistentvolumeclaim *PersistentVolumeClaim) Save() error {
	update, err := persistentvolumeclaim.client.Kubernetes.
		CoreV1().
		PersistentVolumeClaims(persistentvolumeclaim.Namespace).
		Update(persistentvolumeclaim.client.Ctx, &persistentvolumeclaim.PersistentVolumeClaim, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to save persistentvolumeclaim %s in namespace %s: %w", persistentvolumeclaim.Name, persistentvolumeclaim.Namespace, err)
	}

	persistentvolumeclaim.PersistentVolumeClaim = *update

	return nil
}
