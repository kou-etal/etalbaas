package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

func TestClaimsFromRequest(t *testing.T) {
	// No project ID in context.
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	_, _, err := claimsFromRequest(r)
	if err == nil {
		t.Fatal("expected error for missing project_id")
	}

	// With project ID and role.
	ctx := requestctx.WithProjectID(r.Context(), "proj1")
	ctx = requestctx.WithRole(ctx, "service_role")
	r = r.WithContext(ctx)

	projectID, claims, err := claimsFromRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if projectID != "proj1" {
		t.Errorf("projectID = %q, want %q", projectID, "proj1")
	}

	var parsed struct {
		Role string `json:"role"`
		Sub  string `json:"sub"`
	}
	if err := json.Unmarshal(claims, &parsed); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	if parsed.Role != "service_role" {
		t.Errorf("claims role = %q, want %q", parsed.Role, "service_role")
	}
}

func TestClaimsFromRequest_DefaultRole(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := requestctx.WithProjectID(r.Context(), "proj1")
	r = r.WithContext(ctx)

	_, claims, err := claimsFromRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(claims, &parsed); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	if parsed.Role != "authenticated" {
		t.Errorf("default role = %q, want %q", parsed.Role, "authenticated")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	if !strings.Contains(w.Body.String(), `"key":"value"`) {
		t.Errorf("body = %q, missing expected content", w.Body.String())
	}
}

func TestWriteAppError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", apperror.New(apperror.CodeNotFound, "bucket not found"), http.StatusNotFound},
		{"invalid arg", apperror.New(apperror.CodeInvalidArgument, "bad input"), http.StatusBadRequest},
		{"permission denied", apperror.New(apperror.CodePermissionDenied, "forbidden"), http.StatusForbidden},
		{"internal", apperror.New(apperror.CodeInternal, "db error"), http.StatusInternalServerError},
		{"already exists", apperror.New(apperror.CodeAlreadyExists, "duplicate"), http.StatusConflict},
		{"precondition", apperror.New(apperror.CodeFailedPrecondition, "not empty"), http.StatusPreconditionFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeAppError(w, tt.err)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			// Internal errors should be masked.
			if tt.wantStatus == http.StatusInternalServerError {
				if strings.Contains(w.Body.String(), "db error") {
					t.Error("internal error message should be masked")
				}
			}
		})
	}
}

func TestAppErrorToHTTPStatus(t *testing.T) {
	if s := appErrorToHTTPStatus(apperror.CodeNotFound); s != http.StatusNotFound {
		t.Errorf("CodeNotFound → %d, want %d", s, http.StatusNotFound)
	}
	if s := appErrorToHTTPStatus(apperror.CodeUnknown); s != http.StatusInternalServerError {
		t.Errorf("CodeUnknown → %d, want %d", s, http.StatusInternalServerError)
	}
}

func TestUploadHandler_MissingAuth(t *testing.T) {
	handler := &UploadHandler{svc: nil}
	r := httptest.NewRequest(http.MethodPost, "/storage/v1/object/images/cat.png", strings.NewReader("body"))
	r.SetPathValue("bucket", "images")
	r.SetPathValue("path", "cat.png")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestDownloadHandler_MissingAuth(t *testing.T) {
	handler := &DownloadHandler{svc: nil}
	r := httptest.NewRequest(http.MethodGet, "/storage/v1/object/images/cat.png", nil)
	r.SetPathValue("bucket", "images")
	r.SetPathValue("path", "cat.png")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestDeleteHandler_MissingAuth(t *testing.T) {
	handler := &DeleteHandler{svc: nil}
	r := httptest.NewRequest(http.MethodDelete, "/storage/v1/object/images/cat.png", nil)
	r.SetPathValue("bucket", "images")
	r.SetPathValue("path", "cat.png")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestListHandler_MissingAuth(t *testing.T) {
	handler := &ListHandler{svc: nil}
	r := httptest.NewRequest(http.MethodGet, "/storage/v1/object/list/images", nil)
	r.SetPathValue("bucket", "images")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestPublicHandler_MissingProjectID(t *testing.T) {
	handler := &PublicHandler{svc: nil}
	r := httptest.NewRequest(http.MethodGet, "/storage/v1/public/images/cat.png", nil)
	r.SetPathValue("bucket", "images")
	r.SetPathValue("path", "cat.png")
	// No project_id query parameter.

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
