package kubernetes

// Code generated by stub-gen; DO NOT EDIT.

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rosskirkpat/kscaler/pkg/client"
)

// ServiceAccount wraps a Kubernetes ServiceAccount.
type ServiceAccount struct {
	corev1.ServiceAccount

	client client.Client
}

// NewServiceAccount creates a ServiceAccount from its Kubernetes ServiceAccount.
func NewServiceAccount(client client.Client, serviceaccount corev1.ServiceAccount) (ServiceAccount, error) {
	createdServiceAccount, err := client.Kubernetes.
		CoreV1().
		ServiceAccounts(serviceaccount.Namespace).
		Create(client.Ctx, &serviceaccount, metav1.CreateOptions{})
	if err != nil {
		return ServiceAccount{}, fmt.Errorf("failed to create serviceaccount %s in namespace %s: %w", serviceaccount.Name, serviceaccount.Namespace, err)
	}

	return ServiceAccount{
		ServiceAccount: *createdServiceAccount,
		client: client,
	}, nil
}

// GetServiceAccount gets a serviceaccount in a namespace.
func GetServiceAccount(client client.Client, name string, namespace string) (ServiceAccount, error) {
	options := metav1.GetOptions{}

	serviceaccount, err := client.Kubernetes.
		CoreV1().
		ServiceAccounts(namespace).
		Get(client.Ctx, name, options)
	if err != nil {
		return ServiceAccount{}, fmt.Errorf("failed to get serviceaccount %s in namespace %s: %w", name, namespace, err)
	}

	return ServiceAccount{
		ServiceAccount: *serviceaccount,
		client: client,
	}, nil
}

// ListServiceAccounts lists all serviceaccounts in a namespace.
func ListServiceAccounts(client client.Client, namespace string) ([]ServiceAccount, error) {
	options := metav1.ListOptions{}

	list, err := client.Kubernetes.
		CoreV1().
		ServiceAccounts(namespace).
		List(client.Ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list serviceaccounts in namespace %s: %w", namespace, err)
	}

	serviceaccounts := make([]ServiceAccount, 0, len(list.Items))

	for _, item := range list.Items {
		serviceaccounts = append(serviceaccounts, ServiceAccount{
			ServiceAccount: item,
			client: client,
		})
	}

	return serviceaccounts, nil
}

// Delete deletes a ServiceAccount from the Kubernetes cluster.
func (serviceaccount ServiceAccount) Delete() error {
	options := metav1.DeleteOptions{}

	err := serviceaccount.client.Kubernetes.
		CoreV1().
		ServiceAccounts(serviceaccount.Namespace).
		Delete(serviceaccount.client.Ctx, serviceaccount.Name, options)
	if err != nil {
		return fmt.Errorf("failed to delete serviceaccount %s in namespace %s: %w", serviceaccount.Name, serviceaccount.Namespace, err)
	}

	return nil
}

// Update gets the current ServiceAccount status.
func (serviceaccount *ServiceAccount) Update() error {
	options := metav1.GetOptions{}

	update, err := serviceaccount.client.Kubernetes.
		CoreV1().
		ServiceAccounts(serviceaccount.Namespace).
		Get(serviceaccount.client.Ctx, serviceaccount.Name, options)
	if err != nil {
		return fmt.Errorf("failed to update serviceaccount %s in namespace %s: %w", serviceaccount.Name, serviceaccount.Namespace, err)
	}

	serviceaccount.ServiceAccount = *update

	return nil
}

// Save saves the current ServiceAccount.
func (serviceaccount *ServiceAccount) Save() error {
	update, err := serviceaccount.client.Kubernetes.
		CoreV1().
		ServiceAccounts(serviceaccount.Namespace).
		Update(serviceaccount.client.Ctx, &serviceaccount.ServiceAccount, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to save serviceaccount %s in namespace %s: %w", serviceaccount.Name, serviceaccount.Namespace, err)
	}

	serviceaccount.ServiceAccount = *update

	return nil
}
