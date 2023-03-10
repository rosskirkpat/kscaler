package kubernetes

// Code generated by stub-gen; DO NOT EDIT.

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rosskirkpat/kscaler/pkg/client"
)

// Secret wraps a Kubernetes Secret.
type Secret struct {
	corev1.Secret

	client client.Client
}

// NewSecret creates a Secret from its Kubernetes Secret.
func NewSecret(client client.Client, secret corev1.Secret) (Secret, error) {
	createdSecret, err := client.Kubernetes.
		CoreV1().
		Secrets(secret.Namespace).
		Create(client.Ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		return Secret{}, fmt.Errorf("failed to create secret %s in namespace %s: %w", secret.Name, secret.Namespace, err)
	}

	return Secret{
		Secret: *createdSecret,
		client: client,
	}, nil
}

// GetSecret gets a secret in a namespace.
func GetSecret(client client.Client, name string, namespace string) (Secret, error) {
	options := metav1.GetOptions{}

	secret, err := client.Kubernetes.
		CoreV1().
		Secrets(namespace).
		Get(client.Ctx, name, options)
	if err != nil {
		return Secret{}, fmt.Errorf("failed to get secret %s in namespace %s: %w", name, namespace, err)
	}

	return Secret{
		Secret: *secret,
		client: client,
	}, nil
}

// ListSecrets lists all secrets in a namespace.
func ListSecrets(client client.Client, namespace string) ([]Secret, error) {
	options := metav1.ListOptions{}

	list, err := client.Kubernetes.
		CoreV1().
		Secrets(namespace).
		List(client.Ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets in namespace %s: %w", namespace, err)
	}

	secrets := make([]Secret, 0, len(list.Items))

	for _, item := range list.Items {
		secrets = append(secrets, Secret{
			Secret: item,
			client: client,
		})
	}

	return secrets, nil
}

// Delete deletes a Secret from the Kubernetes cluster.
func (secret Secret) Delete() error {
	options := metav1.DeleteOptions{}

	err := secret.client.Kubernetes.
		CoreV1().
		Secrets(secret.Namespace).
		Delete(secret.client.Ctx, secret.Name, options)
	if err != nil {
		return fmt.Errorf("failed to delete secret %s in namespace %s: %w", secret.Name, secret.Namespace, err)
	}

	return nil
}

// Update gets the current Secret status.
func (secret *Secret) Update() error {
	options := metav1.GetOptions{}

	update, err := secret.client.Kubernetes.
		CoreV1().
		Secrets(secret.Namespace).
		Get(secret.client.Ctx, secret.Name, options)
	if err != nil {
		return fmt.Errorf("failed to update secret %s in namespace %s: %w", secret.Name, secret.Namespace, err)
	}

	secret.Secret = *update

	return nil
}

// Save saves the current Secret.
func (secret *Secret) Save() error {
	update, err := secret.client.Kubernetes.
		CoreV1().
		Secrets(secret.Namespace).
		Update(secret.client.Ctx, &secret.Secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to save secret %s in namespace %s: %w", secret.Name, secret.Namespace, err)
	}

	secret.Secret = *update

	return nil
}
