package gpu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const runpodBaseURL = "https://api.runpod.ai/v2"

// maxResponseBodySize is the maximum response body size to read from RunPod API (1 MB).
const maxResponseBodySize = 1 << 20

// RunPodServerlessProvider implements Provider for RunPod Serverless endpoints.
type RunPodServerlessProvider struct {
	apiKeySecret string // K8s Secret name
	httpClient   *http.Client
	k8sClient    kubernetes.Interface
	namespace    string // namespace containing the API key Secret

	// baseURL allows overriding for tests.
	baseURL string

	mu sync.Mutex
	// apiKey is cached after first retrieval from K8s Secret.
	apiKey string
	// endpointID is cached after first Submit for Status/Cancel calls.
	// Can be pre-set via WithRunPodEndpointID (dispatcher use case).
	endpointID string
}

// RunPodServerlessOption configures a RunPodServerlessProvider.
type RunPodServerlessOption func(*RunPodServerlessProvider)

// WithRunPodBaseURL overrides the default RunPod API base URL (for testing).
func WithRunPodBaseURL(url string) RunPodServerlessOption {
	return func(p *RunPodServerlessProvider) {
		p.baseURL = url
	}
}

// WithRunPodAPIKey sets the API key directly (for testing or dispatcher use).
func WithRunPodAPIKey(key string) RunPodServerlessOption {
	return func(p *RunPodServerlessProvider) {
		p.apiKey = key
	}
}

// WithRunPodEndpointID pre-sets the endpoint ID (for dispatcher use).
func WithRunPodEndpointID(id string) RunPodServerlessOption {
	return func(p *RunPodServerlessProvider) {
		p.endpointID = id
	}
}

// NewRunPodServerlessProvider creates a RunPod Serverless provider.
func NewRunPodServerlessProvider(
	apiKeySecret string,
	httpClient *http.Client,
	k8sClient kubernetes.Interface,
	namespace string,
	opts ...RunPodServerlessOption,
) *RunPodServerlessProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	p := &RunPodServerlessProvider{
		apiKeySecret: apiKeySecret,
		httpClient:   httpClient,
		k8sClient:    k8sClient,
		namespace:    namespace,
		baseURL:      runpodBaseURL,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *RunPodServerlessProvider) Name() string                { return "runpod" }
func (p *RunPodServerlessProvider) SupportedProducts() []string { return []string{"serverless"} }
func (p *RunPodServerlessProvider) DataRegion() string          { return "us" }
func (p *RunPodServerlessProvider) SupportedGPUTypes() []string {
	return []string{"A10G", "A100", "H100", "RTX3090", "RTX4090"}
}

// runpodRunRequest is the JSON body for /run.
type runpodRunRequest struct {
	Input json.RawMessage `json:"input"`
}

// runpodRunResponse is the JSON response from /run.
type runpodRunResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// runpodStatusResponse is the JSON response from /status/{job_id}.
type runpodStatusResponse struct {
	ID            string          `json:"id"`
	Status        string          `json:"status"`
	DelayTime     int64           `json:"delayTime"`
	ExecutionTime int64           `json:"executionTime"`
	Output        json.RawMessage `json:"output,omitempty"`
	Error         string          `json:"error,omitempty"`
}

func (p *RunPodServerlessProvider) Submit(ctx context.Context, job GPUJob) (JobID, error) {
	if err := job.Validate(); err != nil {
		return "", err
	}

	endpointID := p.resolveEndpointID(job.ProviderConfig)
	if endpointID == "" {
		return "", fmt.Errorf("RunPod serverless requires provider_config.endpoint_id")
	}

	apiKey, err := p.resolveAPIKey(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve RunPod API key: %w", err)
	}

	input := job.Input
	if input == nil {
		input = []byte("{}")
	}

	body, err := json.Marshal(runpodRunRequest{Input: input})
	if err != nil {
		return "", fmt.Errorf("marshal RunPod request: %w", err)
	}

	reqURL := fmt.Sprintf("%s/%s/run", p.baseURL, url.PathEscape(endpointID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create RunPod request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("RunPod /run request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("RunPod /run returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result runpodRunResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal RunPod response: %w", err)
	}

	if result.ID == "" {
		return "", fmt.Errorf("RunPod /run returned empty job ID")
	}

	return JobID(result.ID), nil
}

func (p *RunPodServerlessProvider) Status(ctx context.Context, id JobID) (JobStatus, error) {
	p.mu.Lock()
	eid := p.endpointID
	p.mu.Unlock()
	if eid == "" {
		return JobStatus{}, fmt.Errorf("endpoint ID not set; call Submit first or use WithRunPodEndpointID")
	}

	apiKey, err := p.resolveAPIKey(ctx)
	if err != nil {
		return JobStatus{}, fmt.Errorf("resolve RunPod API key: %w", err)
	}

	reqURL := fmt.Sprintf("%s/%s/status/%s", p.baseURL, url.PathEscape(eid), url.PathEscape(string(id)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return JobStatus{}, fmt.Errorf("create RunPod status request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return JobStatus{}, fmt.Errorf("RunPod /status request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if resp.StatusCode != http.StatusOK {
		return JobStatus{}, fmt.Errorf("RunPod /status returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result runpodStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return JobStatus{}, fmt.Errorf("unmarshal RunPod status: %w", err)
	}

	return JobStatus{
		State:           mapRunPodState(result.Status),
		DelayTimeMs:     result.DelayTime,
		ExecutionTimeMs: result.ExecutionTime,
		Output:          result.Output,
		Error:           result.Error,
	}, nil
}

func (p *RunPodServerlessProvider) Cancel(ctx context.Context, id JobID) error {
	p.mu.Lock()
	eid := p.endpointID
	p.mu.Unlock()
	if eid == "" {
		return fmt.Errorf("endpoint ID not set; call Submit first or use WithRunPodEndpointID")
	}

	apiKey, err := p.resolveAPIKey(ctx)
	if err != nil {
		return fmt.Errorf("resolve RunPod API key: %w", err)
	}

	reqURL := fmt.Sprintf("%s/%s/cancel/%s", p.baseURL, url.PathEscape(eid), url.PathEscape(string(id)))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create RunPod cancel request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("RunPod /cancel request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
		return fmt.Errorf("RunPod /cancel returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// SubmitSync sends a GPU job to RunPod's /runsync endpoint, which blocks until
// the job completes (or times out on RunPod's side, typically ~120s).
// Unlike Submit+Poll, this avoids polling overhead for short-lived jobs.
// Image validation is skipped because /runsync uses the endpoint's worker image.
func (p *RunPodServerlessProvider) SubmitSync(ctx context.Context, job GPUJob) (JobStatus, error) {
	if job.GPUType == "" {
		return JobStatus{}, fmt.Errorf("gpu job gpu_type must not be empty")
	}

	endpointID := p.resolveEndpointID(job.ProviderConfig)
	if endpointID == "" {
		return JobStatus{}, fmt.Errorf("RunPod serverless requires provider_config.endpoint_id")
	}

	apiKey, err := p.resolveAPIKey(ctx)
	if err != nil {
		return JobStatus{}, fmt.Errorf("resolve RunPod API key: %w", err)
	}

	input := job.Input
	if input == nil {
		input = []byte("{}")
	}

	body, err := json.Marshal(runpodRunRequest{Input: input})
	if err != nil {
		return JobStatus{}, fmt.Errorf("marshal RunPod request: %w", err)
	}

	reqURL := fmt.Sprintf("%s/%s/runsync", p.baseURL, url.PathEscape(endpointID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return JobStatus{}, fmt.Errorf("create RunPod request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return JobStatus{}, fmt.Errorf("RunPod /runsync request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if resp.StatusCode != http.StatusOK {
		return JobStatus{}, fmt.Errorf("RunPod /runsync returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result runpodStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return JobStatus{}, fmt.Errorf("unmarshal RunPod runsync response: %w", err)
	}

	return JobStatus{
		State:           mapRunPodState(result.Status),
		DelayTimeMs:     result.DelayTime,
		ExecutionTimeMs: result.ExecutionTime,
		Output:          result.Output,
		Error:           result.Error,
	}, nil
}

func (p *RunPodServerlessProvider) EstimateCost(_ GPUJob) (Cost, error) {
	// Phase 1: no billing, return zero cost.
	return Cost{Amount: 0, Currency: "USD", Unit: "per-second"}, nil
}

// resolveAPIKey reads the RunPod API key from the K8s Secret.
// The key is cached for the lifetime of the provider instance.
func (p *RunPodServerlessProvider) resolveAPIKey(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.apiKey != "" {
		return p.apiKey, nil
	}
	if p.k8sClient == nil {
		return "", fmt.Errorf("k8s client not available for secret resolution")
	}

	secret, err := p.k8sClient.CoreV1().Secrets(p.namespace).Get(ctx, p.apiKeySecret, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get secret %s: %w", p.apiKeySecret, err)
	}

	key, ok := secret.Data["api-key"]
	if !ok {
		return "", fmt.Errorf("secret %s missing 'api-key' field", p.apiKeySecret)
	}

	p.apiKey = string(key)
	return p.apiKey, nil
}

// resolveEndpointID returns the endpoint ID from ProviderConfig or the pre-set value.
// Caches the endpoint ID for subsequent Status/Cancel calls.
func (p *RunPodServerlessProvider) resolveEndpointID(providerConfig map[string]string) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if id := providerConfig["endpoint_id"]; id != "" {
		p.endpointID = id
		return id
	}
	return p.endpointID
}

func mapRunPodState(status string) JobState {
	switch status {
	case "IN_QUEUE":
		return JobStateQueued
	case "IN_PROGRESS":
		return JobStateRunning
	case "COMPLETED":
		return JobStateCompleted
	case "FAILED", "TIMED_OUT":
		return JobStateFailed
	case "CANCELLED":
		return JobStateCancelled
	default:
		return JobStateQueued
	}
}
