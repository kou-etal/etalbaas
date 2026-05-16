package gpu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRunPodServerless_Name(t *testing.T) {
	p := NewRunPodServerlessProvider("secret", nil, nil, "ns")
	if got := p.Name(); got != "runpod" {
		t.Errorf("Name() = %q, want %q", got, "runpod")
	}
}

func TestRunPodServerless_Submit_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/ep-123/run" {
			t.Errorf("path = %s, want /ep-123/run", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("auth = %q, want %q", auth, "Bearer test-key")
		}

		var body runpodRunRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(runpodRunResponse{
			ID:     "job-abc",
			Status: "IN_QUEUE",
		})
	}))
	defer server.Close()

	p := NewRunPodServerlessProvider("", nil, nil, "",
		WithRunPodBaseURL(server.URL),
		WithRunPodAPIKey("test-key"),
	)

	jobID, err := p.Submit(context.Background(), GPUJob{
		Image:          "myimage:latest",
		GPUType:        "H100",
		Input:          []byte(`{"prompt":"hello"}`),
		ProviderConfig: map[string]string{"endpoint_id": "ep-123"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if jobID != "job-abc" {
		t.Errorf("jobID = %q, want %q", jobID, "job-abc")
	}
}

func TestRunPodServerless_Submit_MissingEndpointID(t *testing.T) {
	p := NewRunPodServerlessProvider("", nil, nil, "",
		WithRunPodAPIKey("test-key"),
	)

	_, err := p.Submit(context.Background(), GPUJob{
		Image:   "myimage:latest",
		GPUType: "H100",
	})
	if err == nil {
		t.Error("expected error for missing endpoint_id")
	}
}

func TestRunPodServerless_Submit_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	p := NewRunPodServerlessProvider("", nil, nil, "",
		WithRunPodBaseURL(server.URL),
		WithRunPodAPIKey("test-key"),
	)

	_, err := p.Submit(context.Background(), GPUJob{
		Image:          "myimage:latest",
		GPUType:        "H100",
		ProviderConfig: map[string]string{"endpoint_id": "ep-123"},
	})
	if err == nil {
		t.Error("expected error for 400 response")
	}
}

func TestRunPodServerless_Status_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/ep-123/status/job-abc" {
			t.Errorf("path = %s, want /ep-123/status/job-abc", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(runpodStatusResponse{
			ID:            "job-abc",
			Status:        "COMPLETED",
			DelayTime:     1500,
			ExecutionTime: 3200,
			Output:        json.RawMessage(`{"result":"done"}`),
		})
	}))
	defer server.Close()

	p := NewRunPodServerlessProvider("", nil, nil, "",
		WithRunPodBaseURL(server.URL),
		WithRunPodAPIKey("test-key"),
		WithRunPodEndpointID("ep-123"),
	)

	status, err := p.Status(context.Background(), "job-abc")
	if err != nil {
		t.Fatal(err)
	}
	if status.State != JobStateCompleted {
		t.Errorf("state = %q, want %q", status.State, JobStateCompleted)
	}
	if status.DelayTimeMs != 1500 {
		t.Errorf("delay = %d, want 1500", status.DelayTimeMs)
	}
	if status.ExecutionTimeMs != 3200 {
		t.Errorf("exec = %d, want 3200", status.ExecutionTimeMs)
	}
}

func TestRunPodServerless_Status_NoEndpointID(t *testing.T) {
	p := NewRunPodServerlessProvider("", nil, nil, "",
		WithRunPodAPIKey("test-key"),
	)

	_, err := p.Status(context.Background(), "job-abc")
	if err == nil {
		t.Error("expected error when endpoint ID not set")
	}
}

func TestRunPodServerless_Cancel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/ep-123/cancel/job-abc" {
			t.Errorf("path = %s, want /ep-123/cancel/job-abc", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewRunPodServerlessProvider("", nil, nil, "",
		WithRunPodBaseURL(server.URL),
		WithRunPodAPIKey("test-key"),
		WithRunPodEndpointID("ep-123"),
	)

	if err := p.Cancel(context.Background(), "job-abc"); err != nil {
		t.Fatal(err)
	}
}

func TestRunPodServerless_ResolveAPIKeyFromSecret(t *testing.T) {
	clientset := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "runpod-creds",
			Namespace: "platform-system",
		},
		Data: map[string][]byte{
			"api-key": []byte("secret-key-from-k8s"),
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer secret-key-from-k8s" {
			t.Errorf("auth = %q, want %q", auth, "Bearer secret-key-from-k8s")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(runpodRunResponse{ID: "job-1", Status: "IN_QUEUE"})
	}))
	defer server.Close()

	p := NewRunPodServerlessProvider("runpod-creds", nil, clientset, "platform-system",
		WithRunPodBaseURL(server.URL),
	)

	_, err := p.Submit(context.Background(), GPUJob{
		Image:          "img:latest",
		GPUType:        "A100",
		ProviderConfig: map[string]string{"endpoint_id": "ep-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunPodServerless_ResolveAPIKey_MissingSecret(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	p := NewRunPodServerlessProvider("nonexistent", nil, clientset, "platform-system")

	_, err := p.Submit(context.Background(), GPUJob{
		Image:          "img:latest",
		GPUType:        "A100",
		ProviderConfig: map[string]string{"endpoint_id": "ep-1"},
	})
	if err == nil {
		t.Error("expected error for missing secret")
	}
}

func TestRunPodServerless_StateMapping(t *testing.T) {
	tests := []struct {
		runpod string
		want   JobState
	}{
		{"IN_QUEUE", JobStateQueued},
		{"IN_PROGRESS", JobStateRunning},
		{"COMPLETED", JobStateCompleted},
		{"FAILED", JobStateFailed},
		{"TIMED_OUT", JobStateFailed},
		{"CANCELLED", JobStateCancelled},
		{"UNKNOWN", JobStateQueued},
	}
	for _, tt := range tests {
		if got := mapRunPodState(tt.runpod); got != tt.want {
			t.Errorf("mapRunPodState(%q) = %q, want %q", tt.runpod, got, tt.want)
		}
	}
}

func TestRunPodServerless_EstimateCost(t *testing.T) {
	p := NewRunPodServerlessProvider("", nil, nil, "")
	cost, err := p.EstimateCost(GPUJob{})
	if err != nil {
		t.Fatal(err)
	}
	if cost.Amount != 0 {
		t.Errorf("Amount = %f, want 0", cost.Amount)
	}
	if cost.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", cost.Currency, "USD")
	}
}

func TestRunPodServerless_SubmitCachesEndpointID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(runpodRunResponse{ID: "job-1"})
	}))
	defer server.Close()

	p := NewRunPodServerlessProvider("", nil, nil, "",
		WithRunPodBaseURL(server.URL),
		WithRunPodAPIKey("key"),
	)

	// Submit sets the endpoint ID
	_, err := p.Submit(context.Background(), GPUJob{
		Image:          "img:latest",
		GPUType:        "H100",
		ProviderConfig: map[string]string{"endpoint_id": "ep-cached"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Status should work without passing endpoint_id again
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ep-cached/status/job-1" {
			t.Errorf("path = %s, want /ep-cached/status/job-1", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(runpodStatusResponse{Status: "IN_QUEUE"})
	}))
	defer server2.Close()

	p.baseURL = server2.URL
	_, err = p.Status(context.Background(), "job-1")
	if err != nil {
		t.Fatal(err)
	}
}
