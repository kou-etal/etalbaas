package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

// claimsFromRequest builds JWT-like claims from the API key context values
// that were set by the APIKeyValidator middleware.
func claimsFromRequest(r *http.Request) (projectID string, claims json.RawMessage, err error) {
	projectID, err = requestctx.ProjectID(r.Context())
	if err != nil {
		return "", nil, fmt.Errorf("project_id not found in context")
	}
	role, _ := requestctx.Role(r.Context())
	if role == "" {
		role = "authenticated"
	}
	// "sub":"service" is a placeholder — API key callers are service-level, not end-users.
	// ownerFromClaims will return an empty UUID since "service" is not a valid UUID,
	// which is correct: objects uploaded via API key have no individual owner.
	claims = json.RawMessage(fmt.Sprintf(`{"role":%q,"sub":"service"}`, role))
	return projectID, claims, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeAppError(w http.ResponseWriter, err error) {
	code := apperror.CodeOf(err)
	status := appErrorToHTTPStatus(code)
	msg := err.Error()
	// Mask internal errors.
	if status == http.StatusInternalServerError {
		msg = "internal error"
	}
	writeError(w, status, msg)
}

// sanitizeFilename escapes characters that could cause Content-Disposition header injection.
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, `\`, `\\`)
	name = strings.ReplaceAll(name, `"`, `\"`)
	return name
}

func appErrorToHTTPStatus(code apperror.Code) int {
	switch code {
	case apperror.CodeNotFound:
		return http.StatusNotFound
	case apperror.CodeAlreadyExists:
		return http.StatusConflict
	case apperror.CodeInvalidArgument:
		return http.StatusBadRequest
	case apperror.CodePermissionDenied:
		return http.StatusForbidden
	case apperror.CodeUnauthenticated:
		return http.StatusUnauthorized
	case apperror.CodeFailedPrecondition:
		return http.StatusPreconditionFailed
	case apperror.CodeResourceExhausted:
		return http.StatusTooManyRequests
	case apperror.CodeCanceled, apperror.CodeDeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
