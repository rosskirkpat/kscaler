package kubernetes

import (
	"github.com/rosskirkpat/kscaler/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:generate stub-gen -api CoreV1 -type Secret

// SecretBuilder tracks the options set for a secret.
type SecretBuilder struct {
	Name       string
	Namespace  string
	Data       map[string][]byte
	StringData map[string]string
}

// CreateSecret creates a secret.
// Additional parameters can be added to this call.
// The creation is started by calling 'Do'.
func CreateSecret(name string) SecretBuilder {
	return SecretBuilder{
		Name: name,
	}
}

// WithNamespace sets the namespace in which the secret will be created.
func (builder SecretBuilder) WithNamespace(namespace string) SecretBuilder {
	builder.Namespace = namespace

	return builder
}

// WithData sets the data the secret should hold.
func (builder SecretBuilder) WithData(data map[string][]byte) SecretBuilder {
	builder.Data = data

	return builder
}

// WithStringData sets the data the secret should hold as string
func (builder SecretBuilder) WithStringData(data map[string]string) SecretBuilder {
	builder.StringData = data

	return builder
}

// Do creates the secret in the cluster.
func (builder SecretBuilder) Do(client client.Client) (Secret, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.Name,
			Namespace: builder.Namespace,
		},

		Data:       builder.Data,
		StringData: builder.StringData,
	}

	return NewSecret(client, secret)
}
