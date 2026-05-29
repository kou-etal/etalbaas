package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/k8s"
	"github.com/kou-etal/etalbaas/services/project/internal/store"
)

const (
	maxDisplayNameLen       = 100
	maxDescriptionLen       = 500
	maxApiKeyNameLen        = 63
	projectIDLen            = 8
	apiKeyRawBytes          = 32
	apiKeyPrefixLen         = 8
	defaultExpiresInDays    = 90
	noExpiryYear            = 9999
	maxProjectsPerTenant    = 2 // Free plan limit
)

var allowedExtensions = map[string]bool{
	"pgvector": true,
	"pgcrypto": true,
}

var allowedRoles = map[string]bool{
	"anon":         true,
	"service_role": true,
}

type ProjectService struct {
	q      store.Querier
	crdMgr k8s.ProjectCRDManager
}

func NewProjectService(q store.Querier, crdMgr k8s.ProjectCRDManager) *ProjectService {
	return &ProjectService{q: q, crdMgr: crdMgr}
}

type CreateProjectParams struct {
	TenantID           uuid.UUID
	DisplayName        string
	Description        string
	PostgresEnabled    bool
	PostgresExtensions []string
	RedisEnabled       bool
	PostgrestEnabled   bool
}

func (s *ProjectService) CreateProject(ctx context.Context, p CreateProjectParams) (store.Project, error) {
	// Validate inputs first (no DB access needed)
	if err := validateDisplayName(p.DisplayName); err != nil {
		return store.Project{}, err
	}
	if len(p.Description) > maxDescriptionLen {
		return store.Project{}, apperror.New(apperror.CodeInvalidArgument, "description exceeds maximum length")
	}
	if len(p.PostgresExtensions) > 0 && !p.PostgresEnabled {
		return store.Project{}, apperror.New(apperror.CodeInvalidArgument, "postgres extensions require postgres to be enabled")
	}
	for _, ext := range p.PostgresExtensions {
		if !allowedExtensions[ext] {
			return store.Project{}, apperror.New(apperror.CodeInvalidArgument, fmt.Sprintf("invalid postgres extension: %s", ext))
		}
	}
	if p.PostgrestEnabled && !p.PostgresEnabled {
		return store.Project{}, apperror.New(apperror.CodeInvalidArgument, "postgrest requires postgres to be enabled")
	}

	// Enforce per-tenant project count limit
	count, err := s.q.CountActiveProjectsByTenantID(ctx, p.TenantID)
	if err != nil {
		return store.Project{}, wrapDBError(err, "count projects")
	}
	if count >= maxProjectsPerTenant {
		return store.Project{}, apperror.New(apperror.CodeResourceExhausted, "project limit reached for current plan")
	}

	projectID, err := generateProjectID()
	if err != nil {
		return store.Project{}, apperror.Wrap(apperror.CodeInternal, "generate project id", err)
	}

	var desc *string
	if p.Description != "" {
		desc = &p.Description
	}

	row, err := s.q.CreateProject(ctx, store.CreateProjectParams{
		ID:                 projectID,
		TenantID:           p.TenantID,
		DisplayName:        p.DisplayName,
		Description:        desc,
		PostgresEnabled:    p.PostgresEnabled,
		PostgresExtensions: p.PostgresExtensions,
		RedisEnabled:       p.RedisEnabled,
		PostgrestEnabled:   p.PostgrestEnabled,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			slog.Warn("project id collision", "project_id", projectID)
			return store.Project{}, apperror.New(apperror.CodeAlreadyExists, "project id collision, please retry")
		}
		return store.Project{}, wrapDBError(err, "create project")
	}

	// CRD creation: best-effort. Failure is logged; Operator will not reconcile until CRD exists.
	if s.crdMgr != nil {
		if err := s.crdMgr.CreateOrUpdate(ctx, k8s.ProjectCRDParams{
			ProjectID:          projectID,
			DisplayName:        p.DisplayName,
			Description:        p.Description,
			PostgresEnabled:    p.PostgresEnabled,
			PostgresExtensions: p.PostgresExtensions,
			RedisEnabled:       p.RedisEnabled,
			PostgrestEnabled:   p.PostgrestEnabled,
		}); err != nil {
			slog.ErrorContext(ctx, "failed to create Project CRD", "project_id", projectID, "error", err)
		}
	}

	return row, nil
}

func (s *ProjectService) GetProject(ctx context.Context, tenantID uuid.UUID, projectID string) (store.Project, error) {
	row, err := s.q.GetProjectByIDAndTenantID(ctx, store.GetProjectByIDAndTenantIDParams{
		ID:       projectID,
		TenantID: tenantID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Project{}, apperror.New(apperror.CodeNotFound, "project not found")
	}
	if err != nil {
		return store.Project{}, wrapDBError(err, "get project")
	}

	// Lazy-sync: if metaDB status is still pending/provisioning, check the
	// K8s CR phase and update metaDB to match.
	if s.crdMgr != nil && (row.Status == "pending" || row.Status == "provisioning") {
		row = s.syncStatusFromCR(ctx, row, tenantID)
	}

	return row, nil
}

// syncStatusFromCR reads the K8s Project CR phase and updates metaDB when
// the CR has reached a terminal state (Ready / Failed).
func (s *ProjectService) syncStatusFromCR(ctx context.Context, row store.Project, tenantID uuid.UUID) store.Project {
	phase, err := s.crdMgr.GetPhase(ctx, row.ID)
	if err != nil {
		slog.WarnContext(ctx, "failed to get CR phase for status sync", "project_id", row.ID, "error", err)
		return row
	}

	var newStatus string
	switch phase {
	case "Ready":
		newStatus = "ready"
	case "Failed":
		newStatus = "failed"
	case "Provisioning":
		if row.Status == "pending" {
			newStatus = "provisioning"
		}
	}

	if newStatus == "" || newStatus == row.Status {
		return row
	}

	updated, err := s.q.UpdateProjectStatus(ctx, store.UpdateProjectStatusParams{
		ID:       row.ID,
		TenantID: tenantID,
		Status:   newStatus,
	})
	if err != nil {
		slog.WarnContext(ctx, "failed to sync project status from CR", "project_id", row.ID, "new_status", newStatus, "error", err)
		return row
	}
	return updated
}

func (s *ProjectService) ListProjects(ctx context.Context, tenantID uuid.UUID, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]store.Project, error) {
	if limit <= 0 {
		limit = 21
	}
	if limit > 101 {
		limit = 101
	}
	var cursorTS interface{} // nil → SQL NULL; pgtype.Timestamptz{Valid:false} via interface{} causes pgx OID resolution failure
	if cursorCreatedAt != nil {
		cursorTS = pgtype.Timestamptz{Time: *cursorCreatedAt, Valid: true}
	}
	var cursorStr *string
	if cursorID != nil {
		cursorStr = cursorID
	}
	rows, err := s.q.ListProjectsByTenantID(ctx, store.ListProjectsByTenantIDParams{
		TenantID:        tenantID,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorStr,
		PageSize:        limit,
	})
	if err != nil {
		return nil, wrapDBError(err, "list projects")
	}
	return rows, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, tenantID uuid.UUID, projectID string) (store.Project, error) {
	row, err := s.q.UpdateProjectStatus(ctx, store.UpdateProjectStatusParams{
		ID:       projectID,
		TenantID: tenantID,
		Status:   "deleted",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Project{}, apperror.New(apperror.CodeNotFound, "project not found")
	}
	if err != nil {
		return store.Project{}, wrapDBError(err, "delete project")
	}

	// CRD deletion: best-effort.
	if s.crdMgr != nil {
		if err := s.crdMgr.Delete(ctx, projectID); err != nil {
			slog.ErrorContext(ctx, "failed to delete Project CRD", "project_id", projectID, "error", err)
		}
	}

	return row, nil
}

func (s *ProjectService) PauseProject(ctx context.Context, tenantID uuid.UUID, projectID string) (store.Project, error) {
	status, err := s.q.GetProjectStatus(ctx, store.GetProjectStatusParams{
		ID:       projectID,
		TenantID: tenantID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Project{}, apperror.New(apperror.CodeNotFound, "project not found")
	}
	if err != nil {
		return store.Project{}, wrapDBError(err, "get project status")
	}
	if status == "paused" {
		return store.Project{}, apperror.New(apperror.CodeFailedPrecondition, "project is already paused")
	}
	if status == "deleted" || status == "deleting" || status == "failed" {
		return store.Project{}, apperror.New(apperror.CodeFailedPrecondition, "project cannot be paused in current state")
	}

	row, err := s.q.UpdateProjectStatus(ctx, store.UpdateProjectStatusParams{
		ID:       projectID,
		TenantID: tenantID,
		Status:   "paused",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Project{}, apperror.New(apperror.CodeNotFound, "project not found")
	}
	if err != nil {
		return store.Project{}, wrapDBError(err, "pause project")
	}
	return row, nil
}

func (s *ProjectService) ResumeProject(ctx context.Context, tenantID uuid.UUID, projectID string) (store.Project, error) {
	status, err := s.q.GetProjectStatus(ctx, store.GetProjectStatusParams{
		ID:       projectID,
		TenantID: tenantID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Project{}, apperror.New(apperror.CodeNotFound, "project not found")
	}
	if err != nil {
		return store.Project{}, wrapDBError(err, "get project status")
	}
	if status != "paused" {
		return store.Project{}, apperror.New(apperror.CodeFailedPrecondition, "project must be in paused state to resume")
	}

	row, err := s.q.UpdateProjectStatus(ctx, store.UpdateProjectStatusParams{
		ID:       projectID,
		TenantID: tenantID,
		Status:   "ready",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Project{}, apperror.New(apperror.CodeNotFound, "project not found")
	}
	if err != nil {
		return store.Project{}, wrapDBError(err, "resume project")
	}
	return row, nil
}

type CreateApiKeyParams struct {
	ProjectID    string
	TenantID     uuid.UUID
	Name         string
	Role         string
	ExpiresInDays *int32
}

type CreateApiKeyResult struct {
	ApiKey store.ApiKey
	RawKey string
}

func (s *ProjectService) CreateApiKey(ctx context.Context, p CreateApiKeyParams) (CreateApiKeyResult, error) {
	if len(p.Name) == 0 {
		return CreateApiKeyResult{}, apperror.New(apperror.CodeInvalidArgument, "api key name must not be empty")
	}
	if len(p.Name) > maxApiKeyNameLen {
		return CreateApiKeyResult{}, apperror.New(apperror.CodeInvalidArgument, "api key name exceeds maximum length")
	}
	if !allowedRoles[p.Role] {
		return CreateApiKeyResult{}, apperror.New(apperror.CodeInvalidArgument, fmt.Sprintf("invalid role: %s", p.Role))
	}
	if p.ExpiresInDays != nil && *p.ExpiresInDays < 0 {
		return CreateApiKeyResult{}, apperror.New(apperror.CodeInvalidArgument, "expires_in_days must not be negative")
	}

	// Verify project ownership
	if _, err := s.q.GetProjectByIDAndTenantID(ctx, store.GetProjectByIDAndTenantIDParams{
		ID:       p.ProjectID,
		TenantID: p.TenantID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return CreateApiKeyResult{}, apperror.New(apperror.CodeNotFound, "project not found")
	} else if err != nil {
		return CreateApiKeyResult{}, wrapDBError(err, "verify project ownership")
	}

	rawKey, keyHash, keyPrefix, err := generateApiKey()
	if err != nil {
		return CreateApiKeyResult{}, apperror.Wrap(apperror.CodeInternal, "generate api key", err)
	}

	expiresAt := computeExpiresAt(p.ExpiresInDays)

	row, err := s.q.CreateApiKey(ctx, store.CreateApiKeyParams{
		ID:        uuid.New(),
		ProjectID: p.ProjectID,
		Name:      p.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Role:      p.Role,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return CreateApiKeyResult{}, apperror.New(apperror.CodeAlreadyExists, "api key name already exists in this project")
		}
		return CreateApiKeyResult{}, wrapDBError(err, "create api key")
	}
	return CreateApiKeyResult{ApiKey: row, RawKey: rawKey}, nil
}

func (s *ProjectService) ListApiKeys(ctx context.Context, tenantID uuid.UUID, projectID string, limit int32, cursorCreatedAt *time.Time, cursorID *uuid.UUID) ([]store.ApiKey, error) {
	// Verify project ownership
	if _, err := s.q.GetProjectByIDAndTenantID(ctx, store.GetProjectByIDAndTenantIDParams{
		ID:       projectID,
		TenantID: tenantID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.New(apperror.CodeNotFound, "project not found")
	} else if err != nil {
		return nil, wrapDBError(err, "verify project ownership")
	}

	if limit <= 0 {
		limit = 21
	}
	if limit > 101 {
		limit = 101
	}
	var cursorTS interface{}
	if cursorCreatedAt != nil {
		cursorTS = pgtype.Timestamptz{Time: *cursorCreatedAt, Valid: true}
	}
	var cursorUUID pgtype.UUID
	if cursorID != nil {
		cursorUUID = pgtype.UUID{Bytes: *cursorID, Valid: true}
	}
	rows, err := s.q.ListApiKeysByProjectID(ctx, store.ListApiKeysByProjectIDParams{
		ProjectID:       projectID,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageSize:        limit,
	})
	if err != nil {
		return nil, wrapDBError(err, "list api keys")
	}
	return rows, nil
}

func (s *ProjectService) RevokeApiKey(ctx context.Context, tenantID uuid.UUID, projectID string, apiKeyID uuid.UUID) (store.ApiKey, error) {
	// Verify project ownership
	if _, err := s.q.GetProjectByIDAndTenantID(ctx, store.GetProjectByIDAndTenantIDParams{
		ID:       projectID,
		TenantID: tenantID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return store.ApiKey{}, apperror.New(apperror.CodeNotFound, "project not found")
	} else if err != nil {
		return store.ApiKey{}, wrapDBError(err, "verify project ownership")
	}

	row, err := s.q.RevokeApiKey(ctx, store.RevokeApiKeyParams{
		ID:        apiKeyID,
		ProjectID: projectID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ApiKey{}, apperror.New(apperror.CodeNotFound, "api key not found or already revoked")
	}
	if err != nil {
		return store.ApiKey{}, wrapDBError(err, "revoke api key")
	}
	return row, nil
}

// generateProjectID generates a random 8-character alphanumeric project ID.
func generateProjectID() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, projectIDLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

// generateApiKey generates a raw API key, its SHA-256 hash, and prefix.
func generateApiKey() (rawKey, keyHash, keyPrefix string, err error) {
	b := make([]byte, apiKeyRawBytes)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("read random bytes: %w", err)
	}
	rawKey = hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(rawKey))
	keyHash = hex.EncodeToString(hash[:])
	keyPrefix = rawKey[:apiKeyPrefixLen]
	return rawKey, keyHash, keyPrefix, nil
}

// computeExpiresAt computes the expiration time from expires_in_days.
// nil → 90 days, 0 → no expiry (9999-12-31), >0 → now + N days.
func computeExpiresAt(expiresInDays *int32) pgtype.Timestamptz {
	if expiresInDays == nil {
		return pgtype.Timestamptz{
			Time:  time.Now().AddDate(0, 0, defaultExpiresInDays),
			Valid: true,
		}
	}
	if *expiresInDays == 0 {
		return pgtype.Timestamptz{
			Time:  time.Date(noExpiryYear, 12, 31, 23, 59, 59, 0, time.UTC),
			Valid: true,
		}
	}
	return pgtype.Timestamptz{
		Time:  time.Now().AddDate(0, 0, int(*expiresInDays)),
		Valid: true,
	}
}

func validateDisplayName(name string) *apperror.AppError {
	if len(name) == 0 {
		return apperror.New(apperror.CodeInvalidArgument, "display_name must not be empty")
	}
	if len(name) > maxDisplayNameLen {
		return apperror.New(apperror.CodeInvalidArgument, "display_name exceeds maximum length")
	}
	return nil
}

// wrapDBError wraps a database error, preserving context cancellation semantics.
func wrapDBError(err error, msg string) *apperror.AppError {
	slog.Error("database error", "msg", msg, "error", err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperror.Wrap(apperror.CodeCanceled, msg, err)
	}
	return apperror.Wrap(apperror.CodeInternal, msg, err)
}
