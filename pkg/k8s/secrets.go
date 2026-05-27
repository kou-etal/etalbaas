package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const k8sTimeout = 10 * time.Second

// SecretManager manages k8s Secrets for user-defined secrets.
type SecretManager interface {
	CreateSecret(ctx context.Context, namespace, name, key, value string) error
	UpdateSecret(ctx context.Context, namespace, name, key, value string) error
	DeleteSecret(ctx context.Context, namespace, name string) error
}

type secretManager struct {
	client kubernetes.Interface
}

// NewSecretManager creates a SecretManager backed by the given k8s client.
func NewSecretManager(client kubernetes.Interface) SecretManager {
	return &secretManager{client: client}
}

func (m *secretManager) CreateSecret(ctx context.Context, namespace, name, key, value string) error {
	ctx, cancel := context.WithTimeout(ctx, k8sTimeout)
	defer cancel()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			key: []byte(value),
		},
		Type: corev1.SecretTypeOpaque,
	}
	_, err := m.client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create k8s secret %s/%s: %w", namespace, name, err)
	}
	return nil
}

func (m *secretManager) UpdateSecret(ctx context.Context, namespace, name, key, value string) error {
	ctx, cancel := context.WithTimeout(ctx, k8sTimeout)
	defer cancel()

	existing, err := m.client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get k8s secret %s/%s: %w", namespace, name, err)
	}
	if existing.Data == nil {
		existing.Data = make(map[string][]byte)
	}
	existing.Data[key] = []byte(value)
	_, err = m.client.CoreV1().Secrets(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update k8s secret %s/%s: %w", namespace, name, err)
	}
	return nil
}

func (m *secretManager) DeleteSecret(ctx context.Context, namespace, name string) error {
	ctx, cancel := context.WithTimeout(ctx, k8sTimeout)
	defer cancel()

	err := m.client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete k8s secret %s/%s: %w", namespace, name, err)
	}
	return nil
}

// ToK8sSecretName converts an environment variable name to a DNS-1123 compatible k8s secret name.
// Example: "OPENAI_API_KEY" → "openai-api-key"
func ToK8sSecretName(name string) string {
	lower := strings.ToLower(name)
	var b strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// ToK8sNamespace converts a project ID to a k8s namespace.
// Example: "xk7a9bc2" → "project-xk7a9bc2"
func ToK8sNamespace(projectID string) string {
	return "project-" + projectID
}
