package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSecretManager_CreateSecret(t *testing.T) {
	client := fake.NewSimpleClientset()
	mgr := NewSecretManager(client)

	err := mgr.CreateSecret(context.Background(), "project-abc123", "openai-api-key", "OPENAI_API_KEY", "sk-test-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	secret, err := client.CoreV1().Secrets("project-abc123").Get(context.Background(), "openai-api-key", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get created secret: %v", err)
	}
	if string(secret.Data["OPENAI_API_KEY"]) != "sk-test-123" {
		t.Fatalf("got value %q, want %q", string(secret.Data["OPENAI_API_KEY"]), "sk-test-123")
	}
	if secret.Type != corev1.SecretTypeOpaque {
		t.Fatalf("got type %v, want Opaque", secret.Type)
	}
}

func TestSecretManager_UpdateSecret(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openai-api-key",
			Namespace: "project-abc123",
		},
		Data: map[string][]byte{
			"OPENAI_API_KEY": []byte("sk-old"),
		},
	})
	mgr := NewSecretManager(client)

	err := mgr.UpdateSecret(context.Background(), "project-abc123", "openai-api-key", "OPENAI_API_KEY", "sk-new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	secret, err := client.CoreV1().Secrets("project-abc123").Get(context.Background(), "openai-api-key", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get updated secret: %v", err)
	}
	if string(secret.Data["OPENAI_API_KEY"]) != "sk-new" {
		t.Fatalf("got value %q, want %q", string(secret.Data["OPENAI_API_KEY"]), "sk-new")
	}
}

func TestSecretManager_UpdateSecret_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	mgr := NewSecretManager(client)

	err := mgr.UpdateSecret(context.Background(), "project-abc123", "nonexistent", "KEY", "val")
	if err == nil {
		t.Fatal("expected error for nonexistent secret")
	}
}

func TestSecretManager_DeleteSecret(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openai-api-key",
			Namespace: "project-abc123",
		},
	})
	mgr := NewSecretManager(client)

	err := mgr.DeleteSecret(context.Background(), "project-abc123", "openai-api-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = client.CoreV1().Secrets("project-abc123").Get(context.Background(), "openai-api-key", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected secret to be deleted")
	}
}

func TestSecretManager_DeleteSecret_Idempotent(t *testing.T) {
	client := fake.NewSimpleClientset()
	mgr := NewSecretManager(client)

	err := mgr.DeleteSecret(context.Background(), "project-abc123", "nonexistent")
	if err != nil {
		t.Fatalf("expected no error for deleting nonexistent secret, got: %v", err)
	}
}

func TestToK8sSecretName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"OPENAI_API_KEY", "openai-api-key"},
		{"STRIPE_WEBHOOK_SECRET", "stripe-webhook-secret"},
		{"MY_VAR", "my-var"},
		{"SIMPLE", "simple"},
		{"A_B_C", "a-b-c"},
		{"_LEADING_UNDERSCORE", "leading-underscore"},
	}
	for _, tt := range tests {
		got := ToK8sSecretName(tt.input)
		if got != tt.want {
			t.Errorf("ToK8sSecretName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToK8sNamespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"xk7a9bc2", "project-xk7a9bc2"},
		{"abc12345", "project-abc12345"},
	}
	for _, tt := range tests {
		got := ToK8sNamespace(tt.input)
		if got != tt.want {
			t.Errorf("ToK8sNamespace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
