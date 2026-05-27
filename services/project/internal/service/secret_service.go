package service

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
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
	maxSecretNameLen        = 64
	maxSecretValueLen       = 65536
	maxSecretDescriptionLen = 1024
)

var secretNameRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

type SecretService struct {
	q      store.Querier
	secMgr k8s.SecretManager
}

func NewSecretService(q store.Querier, secMgr k8s.SecretManager) *SecretService {
	return &SecretService{q: q, secMgr: secMgr}
}

type CreateSecretParams struct {
	TenantID    uuid.UUID
	ProjectID   string
	Name        string
	Value       string
	Description string
}

func (s *SecretService) CreateSecret(ctx context.Context, p CreateSecretParams) (store.SecretsMetadatum, error) {
	if err := validateSecretName(p.Name); err != nil {
		return store.SecretsMetadatum{}, err
	}
	if err := validateSecretValue(p.Value); err != nil {
		return store.SecretsMetadatum{}, err
	}
	if len(p.Description) > maxSecretDescriptionLen {
		return store.SecretsMetadatum{}, apperror.New(apperror.CodeInvalidArgument, "description exceeds maximum length")
	}

	// Verify project ownership
	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return store.SecretsMetadatum{}, err
	}

	namespace := k8s.ToK8sNamespace(p.ProjectID)
	k8sName := k8s.ToK8sSecretName(p.Name)

	// Create k8s Secret first
	if err := s.secMgr.CreateSecret(ctx, namespace, k8sName, p.Name, p.Value); err != nil {
		return store.SecretsMetadatum{}, apperror.Wrap(apperror.CodeInternal, "create k8s secret", err)
	}

	// Then insert DB metadata
	row, err := s.q.CreateSecretMetadata(ctx, store.CreateSecretMetadataParams{
		ID:          uuid.New(),
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
	})
	if err != nil {
		// Best-effort cleanup of k8s Secret on DB failure
		if delErr := s.secMgr.DeleteSecret(ctx, namespace, k8sName); delErr != nil {
			slog.Error("failed to cleanup k8s secret after db error", "error", delErr, "namespace", namespace, "name", k8sName)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return store.SecretsMetadatum{}, apperror.New(apperror.CodeAlreadyExists, "secret name already exists in this project")
		}
		return store.SecretsMetadatum{}, wrapDBError(err, "create secret metadata")
	}
	return row, nil
}

func (s *SecretService) ListSecrets(ctx context.Context, tenantID uuid.UUID, projectID string, limit int32, cursorCreatedAt *time.Time, cursorID *uuid.UUID) ([]store.SecretsMetadatum, error) {
	// Verify project ownership
	if err := s.verifyProjectOwnership(ctx, projectID, tenantID); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 21
	}
	if limit > 101 {
		limit = 101
	}
	cursorTS := pgtype.Timestamptz{}
	if cursorCreatedAt != nil {
		cursorTS = pgtype.Timestamptz{Time: *cursorCreatedAt, Valid: true}
	}
	var cursorUUID pgtype.UUID
	if cursorID != nil {
		cursorUUID = pgtype.UUID{Bytes: *cursorID, Valid: true}
	}
	rows, err := s.q.ListSecretsByProjectID(ctx, store.ListSecretsByProjectIDParams{
		ProjectID:       projectID,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageSize:        limit,
	})
	if err != nil {
		return nil, wrapDBError(err, "list secrets")
	}
	return rows, nil
}

func (s *SecretService) UpdateSecretValue(ctx context.Context, tenantID uuid.UUID, projectID string, secretID uuid.UUID, value string) (store.SecretsMetadatum, error) {
	if err := validateSecretValue(value); err != nil {
		return store.SecretsMetadatum{}, err
	}

	// Verify project ownership
	if err := s.verifyProjectOwnership(ctx, projectID, tenantID); err != nil {
		return store.SecretsMetadatum{}, err
	}

	// Get metadata to find the secret name
	meta, err := s.q.GetSecretMetadataByID(ctx, store.GetSecretMetadataByIDParams{
		ID:        secretID,
		ProjectID: projectID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.SecretsMetadatum{}, apperror.New(apperror.CodeNotFound, "secret not found")
	}
	if err != nil {
		return store.SecretsMetadatum{}, wrapDBError(err, "get secret metadata")
	}

	namespace := k8s.ToK8sNamespace(projectID)
	k8sName := k8s.ToK8sSecretName(meta.Name)

	if err := s.secMgr.UpdateSecret(ctx, namespace, k8sName, meta.Name, value); err != nil {
		return store.SecretsMetadatum{}, apperror.Wrap(apperror.CodeInternal, "update k8s secret", err)
	}

	row, err := s.q.UpdateSecretMetadataUpdatedAt(ctx, store.UpdateSecretMetadataUpdatedAtParams{
		ID:        secretID,
		ProjectID: projectID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.SecretsMetadatum{}, apperror.New(apperror.CodeNotFound, "secret not found")
	}
	if err != nil {
		return store.SecretsMetadatum{}, wrapDBError(err, "update secret metadata")
	}
	return row, nil
}

func (s *SecretService) DeleteSecret(ctx context.Context, tenantID uuid.UUID, projectID string, secretID uuid.UUID) (store.SecretsMetadatum, error) {
	// Verify project ownership
	if err := s.verifyProjectOwnership(ctx, projectID, tenantID); err != nil {
		return store.SecretsMetadatum{}, err
	}

	// Get metadata to find the secret name
	meta, err := s.q.GetSecretMetadataByID(ctx, store.GetSecretMetadataByIDParams{
		ID:        secretID,
		ProjectID: projectID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.SecretsMetadatum{}, apperror.New(apperror.CodeNotFound, "secret not found")
	}
	if err != nil {
		return store.SecretsMetadatum{}, wrapDBError(err, "get secret metadata")
	}

	namespace := k8s.ToK8sNamespace(projectID)
	k8sName := k8s.ToK8sSecretName(meta.Name)

	// Delete k8s Secret first
	if err := s.secMgr.DeleteSecret(ctx, namespace, k8sName); err != nil {
		return store.SecretsMetadatum{}, apperror.Wrap(apperror.CodeInternal, "delete k8s secret", err)
	}

	// Then physical-delete DB metadata
	row, err := s.q.DeleteSecretMetadata(ctx, store.DeleteSecretMetadataParams{
		ID:        secretID,
		ProjectID: projectID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.SecretsMetadatum{}, apperror.New(apperror.CodeNotFound, "secret not found")
	}
	if err != nil {
		return store.SecretsMetadatum{}, wrapDBError(err, "delete secret metadata")
	}
	return row, nil
}

func (s *SecretService) verifyProjectOwnership(ctx context.Context, projectID string, tenantID uuid.UUID) error {
	if _, err := s.q.GetProjectByIDAndTenantID(ctx, store.GetProjectByIDAndTenantIDParams{
		ID:       projectID,
		TenantID: tenantID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return apperror.New(apperror.CodeNotFound, "project not found")
	} else if err != nil {
		return wrapDBError(err, "verify project ownership")
	}
	return nil
}

func validateSecretName(name string) *apperror.AppError {
	if len(name) == 0 {
		return apperror.New(apperror.CodeInvalidArgument, "secret name must not be empty")
	}
	if len(name) > maxSecretNameLen {
		return apperror.New(apperror.CodeInvalidArgument, "secret name exceeds maximum length")
	}
	if !secretNameRe.MatchString(name) {
		return apperror.New(apperror.CodeInvalidArgument, "secret name must start with a letter and match [A-Z][A-Z0-9_]*")
	}
	return nil
}

func validateSecretValue(value string) *apperror.AppError {
	if len(value) == 0 {
		return apperror.New(apperror.CodeInvalidArgument, "secret value must not be empty")
	}
	if len(value) > maxSecretValueLen {
		return apperror.New(apperror.CodeInvalidArgument, "secret value exceeds maximum length")
	}
	return nil
}
