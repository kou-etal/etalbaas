package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHashKey(t *testing.T) {
	raw := "test-api-key-12345"
	got := HashKey(raw)
	h := sha256.Sum256([]byte(raw))
	want := hex.EncodeToString(h[:])
	if got != want {
		t.Errorf("HashKey(%q) = %q, want %q", raw, got, want)
	}
}

func TestExtractAPIKey_Header(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("apikey", "my-key")

	got := extractAPIKey(r)
	if got != "my-key" {
		t.Errorf("extractAPIKey() = %q, want %q", got, "my-key")
	}
}

func TestExtractAPIKey_Bearer(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer my-bearer-key")

	got := extractAPIKey(r)
	if got != "my-bearer-key" {
		t.Errorf("extractAPIKey() = %q, want %q", got, "my-bearer-key")
	}
}

func TestExtractAPIKey_Missing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	got := extractAPIKey(r)
	if got != "" {
		t.Errorf("extractAPIKey() = %q, want empty", got)
	}
}

func TestExtractAPIKey_HeaderPriority(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("apikey", "header-key")
	r.Header.Set("Authorization", "Bearer bearer-key")

	got := extractAPIKey(r)
	if got != "header-key" {
		t.Errorf("extractAPIKey() = %q, want %q (apikey header takes priority)", got, "header-key")
	}
}

func TestMiddleware_MissingKey(t *testing.T) {
	v := &APIKeyValidator{pool: nil} // pool won't be called
	handler := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}
