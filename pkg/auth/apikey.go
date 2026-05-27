package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

const apiKeyHeader = "apikey"

var (
	ErrMissingAPIKey = errors.New("missing api key")
	ErrInvalidAPIKey = errors.New("invalid or expired api key")
)

// APIKeyValidator validates API keys against the meta DB api_keys table.
// Uses raw pgx queries (no sqlc) to avoid pkg → services dependency.
type APIKeyValidator struct {
	pool *pgxpool.Pool
}

// NewAPIKeyValidator creates a validator backed by the meta DB.
func NewAPIKeyValidator(pool *pgxpool.Pool) *APIKeyValidator {
	return &APIKeyValidator{pool: pool}
}

// Validate checks a raw API key against the meta DB.
// Returns the associated project_id and role on success.
func (v *APIKeyValidator) Validate(ctx context.Context, rawKey string) (projectID, role string, err error) {
	hash := hashAPIKey(rawKey)
	err = v.pool.QueryRow(ctx,
		`SELECT project_id, role FROM api_keys
		 WHERE key_hash = $1 AND revoked_at IS NULL
		   AND (expires_at IS NULL OR expires_at > now())`,
		hash,
	).Scan(&projectID, &role)
	if err != nil {
		return "", "", ErrInvalidAPIKey
	}
	return projectID, role, nil
}

// Middleware returns an HTTP middleware that validates the apikey header
// and injects ProjectID and Role into the request context.
func (v *APIKeyValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawKey := extractAPIKey(r)
		if rawKey == "" {
			http.Error(w, `{"error":"missing api key"}`, http.StatusUnauthorized)
			return
		}

		projectID, role, err := v.Validate(r.Context(), rawKey)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired api key"}`, http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = requestctx.WithProjectID(ctx, projectID)
		ctx = requestctx.WithRole(ctx, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractAPIKey extracts the API key from request headers.
// Checks "apikey" header first, then "Authorization: Bearer {key}".
func extractAPIKey(r *http.Request) string {
	if key := r.Header.Get(apiKeyHeader); key != "" {
		return key
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func hashAPIKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// HashKey computes the SHA-256 hash of a raw API key (exported for testing).
func HashKey(raw string) string {
	return hashAPIKey(raw)
}
