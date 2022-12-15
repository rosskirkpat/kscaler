package kubernetes

import (
	"context"
	"testing"

	"github.com/rosskirkpat/kscaler/pkg/client"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPod(t *testing.T) {
	const namespace = "test"

	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: namespace,
		},
	}

	c := client.Client{
		Ctx:        context.TODO(),
		Kubernetes: fake.NewSimpleClientset(testPod.DeepCopyObject()),
	}

	pod, err := GetPod(c, testPod.Name, namespace)
	assert.NoError(t, err)
	assert.Equal(t, Pod{
		Pod:    testPod,
		client: c,
	}, pod)

	pods, err := ListPods(c, namespace)
	assert.NoError(t, err)
	assert.Contains(t, pods, pod)
}
