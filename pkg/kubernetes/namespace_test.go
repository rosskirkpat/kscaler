package kubernetes

import (
	"context"
	"testing"

	"github.com/rosskirkpat/kscaler/pkg/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNamespace(t *testing.T) {
	c := client.Client{
		Ctx:        context.TODO(),
		Kubernetes: fake.NewSimpleClientset(),
	}

	const namespace = "test"

	_, err := c.Kubernetes.CoreV1().Namespaces().Get(c.Ctx, namespace, metav1.GetOptions{})
	assert.Error(t, err)

	err = DeleteNamespace(c, namespace)
	assert.Error(t, err)

	err = CreateNamespace(c, namespace)
	assert.NoError(t, err)

	_, err = c.Kubernetes.CoreV1().Namespaces().Get(c.Ctx, namespace, metav1.GetOptions{})
	assert.NoError(t, err)

	err = DeleteNamespace(c, namespace)
	assert.NoError(t, err)

	_, err = c.Kubernetes.CoreV1().Namespaces().Get(c.Ctx, namespace, metav1.GetOptions{})
	assert.Error(t, err)
}
