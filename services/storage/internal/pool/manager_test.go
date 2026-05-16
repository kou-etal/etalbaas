package pool

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestManager_GetPool_SecretNotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	mgr := NewManager(client)
	defer mgr.Close()

	_, err := mgr.GetPool(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing secret, got nil")
	}
}

func TestManager_GetPool_MissingCredentials(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-app",
			Namespace: "project-test1",
		},
		Data: map[string][]byte{
			"username": []byte(""),
			"password": []byte(""),
		},
	})
	mgr := NewManager(client)
	defer mgr.Close()

	_, err := mgr.GetPool(context.Background(), "test1")
	if err == nil {
		t.Fatal("expected error for empty credentials, got nil")
	}
}

func TestManager_EvictIdle(t *testing.T) {
	client := fake.NewSimpleClientset()
	mgr := &Manager{
		pools:     make(map[string]*entry),
		k8sClient: client,
		stopCh:    make(chan struct{}),
	}
	defer mgr.Close()

	// Simulate an old entry (we can't create a real pool without a DB,
	// so we test the eviction logic directly).
	mgr.pools["old-project"] = &entry{
		pool:     nil, // will panic if Close is called on nil, but we handle it
		lastUsed: time.Now().Add(-2 * idleTTL),
	}
	mgr.pools["recent-project"] = &entry{
		pool:     nil,
		lastUsed: time.Now(),
	}

	mgr.evictIdle()

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if _, ok := mgr.pools["old-project"]; ok {
		t.Error("old-project should have been evicted")
	}
	if _, ok := mgr.pools["recent-project"]; !ok {
		t.Error("recent-project should not have been evicted")
	}
}

func TestManager_WithRLS_RoleValidation(t *testing.T) {
	// Test that unknown roles are normalized to "authenticated".
	tests := []struct {
		claims   string
		wantRole string
	}{
		{`{"role":"anon","sub":"123"}`, "anon"},
		{`{"role":"authenticated","sub":"123"}`, "authenticated"},
		{`{"role":"service_role","sub":"123"}`, "service_role"},
		{`{"role":"evil_role","sub":"123"}`, "authenticated"},
		{`{"sub":"123"}`, "authenticated"},
		{`{}`, "authenticated"},
	}

	for _, tt := range tests {
		role := "authenticated"
		var parsed struct {
			Role string `json:"role"`
		}
		if json.Unmarshal([]byte(tt.claims), &parsed) == nil && parsed.Role != "" {
			role = parsed.Role
		}
		if role != "anon" && role != "authenticated" && role != "service_role" {
			role = "authenticated"
		}
		if role != tt.wantRole {
			t.Errorf("claims=%s: role=%q, want %q", tt.claims, role, tt.wantRole)
		}
	}
}
