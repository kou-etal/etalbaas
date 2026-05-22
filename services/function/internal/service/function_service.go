package service

import (
	"context"
	"encoding/json"
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
	"github.com/kou-etal/etalbaas/services/function/internal/store"
)

const (
	minFunctionNameLen = 3
	maxFunctionNameLen = 40
	maxDisplayNameLen  = 100

	maxTimeoutLight    = 30
	maxTimeoutHeavyCPU = 3600
	defaultTimeoutSec  = 30

	maxInlineSourceBytes = 1024 * 1024 // 1MB, matches ConfigMap limit
)

var functionNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

var allowedKinds = map[string]bool{
	"heavy-job":        true,
	"heavy-deployment": true,
	"light-deployment": true,
}

var allowedModes = map[string]bool{
	"sync":   true,
	"async":  true,
	"stream": true,
}

// --- JSONB Go types ---

type GitSourceConfig struct {
	RepoURL string `json:"repo_url"`
	Branch  string `json:"branch"`
	Subpath string `json:"subpath"`
}

type InlineSourceConfig struct {
	Code     string `json:"code"`
	Filename string `json:"filename"`
}

type GpuConfigJSON struct {
	Type           string            `json:"type"`
	Provider       string            `json:"provider"`
	Product        string            `json:"product"`
	ProviderConfig map[string]string `json:"provider_config,omitempty"`
}

type TriggerJSON struct {
	Type           string                      `json:"type"`
	DatabaseChange *DatabaseChangeTriggerJSON   `json:"database_change,omitempty"`
	ObjectStorage  *ObjectStorageTriggerJSON    `json:"object_storage,omitempty"`
}

type DatabaseChangeTriggerJSON struct {
	Table          string   `json:"table"`
	Events         []string `json:"events"`
	Filter         string   `json:"filter,omitempty"`
	IncludeColumns []string `json:"include_columns,omitempty"`
}

type ObjectStorageTriggerJSON struct {
	Bucket string   `json:"bucket"`
	Events []string `json:"events"`
	Prefix string   `json:"prefix,omitempty"`
}

type EnvVarJSON struct {
	Name       string `json:"name"`
	Value      string `json:"value,omitempty"`
	SecretName string `json:"secret_name,omitempty"`
}

// --- Service ---

type FunctionService struct {
	q      store.Querier
	crdMgr k8s.FunctionCRDManager // nil when K8s is disabled
}

func NewFunctionService(q store.Querier, crdMgr k8s.FunctionCRDManager) *FunctionService {
	return &FunctionService{q: q, crdMgr: crdMgr}
}

// --- Create ---

type CreateFunctionParams struct {
	TenantID    uuid.UUID
	ProjectID   string
	Name        string
	DisplayName string
	Kind        string
	Mode        string

	SourceType        string
	SourceConfig      []byte
	SourceStoragePath *string

	RuntimePreset       *string
	RuntimeRequirements []string
	RuntimeDockerfile   *string

	TimeoutSec int32
	GpuConfig  []byte
	Triggers   []byte
	EnvVars    []byte
}

func (s *FunctionService) CreateFunction(ctx context.Context, p CreateFunctionParams) (store.Function, error) {
	if err := validateFunctionName(p.Name); err != nil {
		return store.Function{}, err
	}
	if err := validateDisplayName(p.DisplayName); err != nil {
		return store.Function{}, err
	}
	if err := validateKind(p.Kind); err != nil {
		return store.Function{}, err
	}
	if err := validateMode(p.Mode); err != nil {
		return store.Function{}, err
	}
	if err := validateRuntime(p.RuntimePreset, p.RuntimeDockerfile); err != nil {
		return store.Function{}, err
	}
	if err := validateTriggers(p.Triggers); err != nil {
		return store.Function{}, err
	}
	if err := validateInlineSourceSize(p.SourceType, p.SourceConfig); err != nil {
		return store.Function{}, err
	}

	p.TimeoutSec = applyDefaultTimeout(p.TimeoutSec)
	if err := validateTimeout(p.TimeoutSec, p.Kind, p.GpuConfig); err != nil {
		return store.Function{}, err
	}

	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return store.Function{}, err
	}

	row, err := s.q.CreateFunction(ctx, store.CreateFunctionParams{
		ID:                  uuid.New(),
		ProjectID:           p.ProjectID,
		Name:                p.Name,
		DisplayName:         p.DisplayName,
		Kind:                p.Kind,
		Mode:                p.Mode,
		SourceType:          p.SourceType,
		SourceConfig:        p.SourceConfig,
		SourceStoragePath:   p.SourceStoragePath,
		RuntimePreset:       p.RuntimePreset,
		RuntimeRequirements: p.RuntimeRequirements,
		RuntimeDockerfile:   p.RuntimeDockerfile,
		TimeoutSec:          p.TimeoutSec,
		GpuConfig:           p.GpuConfig,
		Triggers:            p.Triggers,
		EnvVars:             p.EnvVars,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return store.Function{}, apperror.New(apperror.CodeAlreadyExists, "function name already exists in this project")
		}
		return store.Function{}, wrapDBError(err, "create function")
	}

	// CRD creation: best-effort. Failure is logged; next GetFunction will self-heal.
	s.applyCRD(ctx, row)

	return row, nil
}

// --- Get ---

func (s *FunctionService) GetFunction(ctx context.Context, tenantID uuid.UUID, projectID string, functionID uuid.UUID) (store.Function, error) {
	if err := s.verifyProjectOwnership(ctx, projectID, tenantID); err != nil {
		return store.Function{}, err
	}

	row, err := s.q.GetFunctionByIDAndProjectID(ctx, store.GetFunctionByIDAndProjectIDParams{
		ID:        functionID,
		ProjectID: projectID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Function{}, apperror.New(apperror.CodeNotFound, "function not found")
	}
	if err != nil {
		return store.Function{}, wrapDBError(err, "get function")
	}

	// Lazy sync: fetch CRD status and update meta DB if changed.
	row = s.syncBuildStatus(ctx, row)

	return row, nil
}

// --- List ---

func (s *FunctionService) ListFunctions(ctx context.Context, tenantID uuid.UUID, projectID string, limit int32, cursorCreatedAt *time.Time, cursorID *uuid.UUID) ([]store.Function, error) {
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
	rows, err := s.q.ListFunctionsByProjectID(ctx, store.ListFunctionsByProjectIDParams{
		ProjectID:       projectID,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageSize:        limit,
	})
	if err != nil {
		return nil, wrapDBError(err, "list functions")
	}
	return rows, nil
}

// --- Update (read-modify-write) ---

type UpdateFunctionParams struct {
	TenantID  uuid.UUID
	ProjectID string
	ID        uuid.UUID

	DisplayName *string
	Mode        *string

	// Source: if SourceType is non-nil, all 3 source fields are replaced.
	SourceType        *string
	SourceConfig      []byte
	SourceStoragePath *string

	// Runtime: if either is non-nil, both fields are replaced.
	RuntimePreset       *string
	RuntimeRequirements []string
	RuntimeDockerfile   *string
	RuntimeSet          bool // true if runtime oneof was provided in the request

	TimeoutSec *int32
	GpuConfig  []byte
	GpuConfigSet bool // true if gpu_config was provided in the request

	// Repeated fields: always fully replaced (proto3 cannot distinguish "not sent" from "empty").
	Triggers    []byte
	TriggersSet bool
	EnvVars     []byte
	EnvVarsSet  bool
}

func (s *FunctionService) UpdateFunction(ctx context.Context, p UpdateFunctionParams) (store.Function, error) {
	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return store.Function{}, err
	}

	current, err := s.q.GetFunctionByIDAndProjectID(ctx, store.GetFunctionByIDAndProjectIDParams{
		ID:        p.ID,
		ProjectID: p.ProjectID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Function{}, apperror.New(apperror.CodeNotFound, "function not found")
	}
	if err != nil {
		return store.Function{}, wrapDBError(err, "get function for update")
	}

	merged := mergeFunction(current, p)

	if err := validateDisplayName(merged.DisplayName); err != nil {
		return store.Function{}, err
	}
	if err := validateMode(merged.Mode); err != nil {
		return store.Function{}, err
	}
	if err := validateRuntime(merged.RuntimePreset, merged.RuntimeDockerfile); err != nil {
		return store.Function{}, err
	}
	if err := validateTriggers(merged.Triggers); err != nil {
		return store.Function{}, err
	}
	if err := validateInlineSourceSize(merged.SourceType, merged.SourceConfig); err != nil {
		return store.Function{}, err
	}

	timeout := applyDefaultTimeout(merged.TimeoutSec)
	if err := validateTimeout(timeout, current.Kind, merged.GpuConfig); err != nil {
		return store.Function{}, err
	}

	row, err := s.q.UpdateFunction(ctx, store.UpdateFunctionParams{
		ID:                  p.ID,
		ProjectID:           p.ProjectID,
		DisplayName:         merged.DisplayName,
		Mode:                merged.Mode,
		SourceType:          merged.SourceType,
		SourceConfig:        merged.SourceConfig,
		SourceStoragePath:   merged.SourceStoragePath,
		RuntimePreset:       merged.RuntimePreset,
		RuntimeRequirements: merged.RuntimeRequirements,
		RuntimeDockerfile:   merged.RuntimeDockerfile,
		TimeoutSec:          timeout,
		GpuConfig:           merged.GpuConfig,
		Triggers:            merged.Triggers,
		EnvVars:             merged.EnvVars,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Function{}, apperror.New(apperror.CodeNotFound, "function not found")
	}
	if err != nil {
		return store.Function{}, wrapDBError(err, "update function")
	}

	// CRD update: best-effort.
	s.applyCRD(ctx, row)

	return row, nil
}

// mergeFunction produces a merged Function by applying update params to current state.
func mergeFunction(current store.Function, p UpdateFunctionParams) store.Function {
	merged := current

	if p.DisplayName != nil {
		merged.DisplayName = *p.DisplayName
	}
	if p.Mode != nil {
		merged.Mode = *p.Mode
	}

	// Source oneof: if provided, replace all 3 columns
	if p.SourceType != nil {
		merged.SourceType = *p.SourceType
		merged.SourceConfig = p.SourceConfig
		merged.SourceStoragePath = p.SourceStoragePath
	}

	// Runtime oneof: if provided, replace both fields
	if p.RuntimeSet {
		merged.RuntimePreset = p.RuntimePreset
		merged.RuntimeRequirements = p.RuntimeRequirements
		merged.RuntimeDockerfile = p.RuntimeDockerfile
	}

	if p.TimeoutSec != nil {
		merged.TimeoutSec = *p.TimeoutSec
	}

	if p.GpuConfigSet {
		merged.GpuConfig = p.GpuConfig
	}

	// Repeated fields: always fully replaced when set
	if p.TriggersSet {
		merged.Triggers = p.Triggers
	}
	if p.EnvVarsSet {
		merged.EnvVars = p.EnvVars
	}

	return merged
}

// --- Delete (soft delete) ---

func (s *FunctionService) DeleteFunction(ctx context.Context, tenantID uuid.UUID, projectID string, functionID uuid.UUID) (store.Function, error) {
	if err := s.verifyProjectOwnership(ctx, projectID, tenantID); err != nil {
		return store.Function{}, err
	}

	row, err := s.q.DeleteFunction(ctx, store.DeleteFunctionParams{
		ID:        functionID,
		ProjectID: projectID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Function{}, apperror.New(apperror.CodeNotFound, "function not found")
	}
	if err != nil {
		return store.Function{}, wrapDBError(err, "delete function")
	}

	// CRD deletion: best-effort.
	if s.crdMgr != nil {
		if err := s.crdMgr.Delete(ctx, projectID, row.Name); err != nil {
			slog.ErrorContext(ctx, "failed to delete Function CRD", "function", row.Name, "error", err)
		}
	}

	return row, nil
}

// --- List Invocations ---

type ListInvocationsParams struct {
	TenantID   uuid.UUID
	ProjectID  string
	FunctionID uuid.UUID

	StatusFilter *string
	Since        *time.Time
	Until        *time.Time

	Limit           int32
	CursorCreatedAt *time.Time
	CursorID        *uuid.UUID
}

func (s *FunctionService) ListInvocations(ctx context.Context, p ListInvocationsParams) ([]store.Invocation, error) {
	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return nil, err
	}

	// Verify function exists
	if _, err := s.q.GetFunctionByIDAndProjectID(ctx, store.GetFunctionByIDAndProjectIDParams{
		ID:        p.FunctionID,
		ProjectID: p.ProjectID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.New(apperror.CodeNotFound, "function not found")
	} else if err != nil {
		return nil, wrapDBError(err, "get function for invocations")
	}

	if p.Limit <= 0 {
		p.Limit = 21
	}
	if p.Limit > 101 {
		p.Limit = 101
	}

	var since, until pgtype.Timestamptz
	if p.Since != nil {
		since = pgtype.Timestamptz{Time: *p.Since, Valid: true}
	}
	if p.Until != nil {
		until = pgtype.Timestamptz{Time: *p.Until, Valid: true}
	}

	cursorTS := pgtype.Timestamptz{}
	if p.CursorCreatedAt != nil {
		cursorTS = pgtype.Timestamptz{Time: *p.CursorCreatedAt, Valid: true}
	}
	var cursorUUID pgtype.UUID
	if p.CursorID != nil {
		cursorUUID = pgtype.UUID{Bytes: *p.CursorID, Valid: true}
	}

	rows, err := s.q.ListInvocationsByFunctionID(ctx, store.ListInvocationsByFunctionIDParams{
		FunctionID:      p.FunctionID,
		ProjectID:       p.ProjectID,
		StatusFilter:    p.StatusFilter,
		Since:           since,
		Until:           until,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageSize:        p.Limit,
	})
	if err != nil {
		return nil, wrapDBError(err, "list invocations")
	}
	return rows, nil
}

// --- Validation ---

func validateFunctionName(name string) *apperror.AppError {
	if len(name) < minFunctionNameLen {
		return apperror.New(apperror.CodeInvalidArgument, "function name must be at least 3 characters")
	}
	if len(name) > maxFunctionNameLen {
		return apperror.New(apperror.CodeInvalidArgument, "function name exceeds maximum length")
	}
	if !functionNameRe.MatchString(name) {
		return apperror.New(apperror.CodeInvalidArgument, "function name must match [a-z0-9][a-z0-9-]*[a-z0-9]")
	}
	return nil
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

func validateKind(kind string) *apperror.AppError {
	if !allowedKinds[kind] {
		return apperror.New(apperror.CodeInvalidArgument, "invalid kind: must be heavy-job, heavy-deployment, or light-deployment")
	}
	return nil
}

func validateMode(mode string) *apperror.AppError {
	if !allowedModes[mode] {
		return apperror.New(apperror.CodeInvalidArgument, "invalid mode: must be sync, async, or stream")
	}
	return nil
}

func validateRuntime(preset *string, dockerfile *string) *apperror.AppError {
	hasPreset := preset != nil && *preset != ""
	hasDockerfile := dockerfile != nil && *dockerfile != ""
	if hasPreset && hasDockerfile {
		return apperror.New(apperror.CodeInvalidArgument, "runtime must be either preset or custom, not both")
	}
	if !hasPreset && !hasDockerfile {
		return apperror.New(apperror.CodeInvalidArgument, "runtime must be specified: either preset or custom")
	}
	return nil
}

func validateTriggers(triggersJSON []byte) *apperror.AppError {
	if len(triggersJSON) == 0 {
		return nil
	}
	var triggers []TriggerJSON
	if err := json.Unmarshal(triggersJSON, &triggers); err != nil {
		return apperror.New(apperror.CodeInvalidArgument, "invalid triggers format")
	}
	for _, t := range triggers {
		switch t.Type {
		case "database_change", "object_storage":
			// allowed
		default:
			return apperror.New(apperror.CodeInvalidArgument, "unsupported trigger type: only database_change and object_storage are allowed in Phase 1")
		}
	}
	return nil
}

func applyDefaultTimeout(timeout int32) int32 {
	if timeout <= 0 {
		return defaultTimeoutSec
	}
	return timeout
}

func hasValidGPUConfig(gpuConfig []byte) bool {
	if len(gpuConfig) == 0 || string(gpuConfig) == "null" {
		return false
	}
	var gc GpuConfigJSON
	if err := json.Unmarshal(gpuConfig, &gc); err != nil {
		return false
	}
	return gc.Type != ""
}

func validateTimeout(timeout int32, kind string, gpuConfig []byte) *apperror.AppError {
	hasGPU := hasValidGPUConfig(gpuConfig)

	switch kind {
	case "light-deployment":
		if timeout > maxTimeoutLight {
			return apperror.New(apperror.CodeInvalidArgument, "timeout for light-deployment must not exceed 30 seconds")
		}
	case "heavy-job", "heavy-deployment":
		if !hasGPU && timeout > maxTimeoutHeavyCPU {
			return apperror.New(apperror.CodeInvalidArgument, "timeout for heavy-* without GPU must not exceed 3600 seconds")
		}
		// GPU: no upper limit
	}
	return nil
}

// --- Helpers ---

func (s *FunctionService) verifyProjectOwnership(ctx context.Context, projectID string, tenantID uuid.UUID) error {
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

func wrapDBError(err error, msg string) *apperror.AppError {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperror.Wrap(apperror.CodeCanceled, msg, err)
	}
	return apperror.Wrap(apperror.CodeInternal, msg, err)
}

// --- CRD integration ---

// applyCRD creates or updates the Function CRD. Failures are logged only.
func (s *FunctionService) applyCRD(ctx context.Context, row store.Function) {
	if s.crdMgr == nil {
		return
	}
	params, err := buildCRDParamsFromRow(row)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build CRD params from DB row",
			"function", row.Name, "project_id", row.ProjectID, "error", err)
		return
	}
	if err := s.crdMgr.CreateOrUpdate(ctx, params); err != nil {
		slog.ErrorContext(ctx, "failed to apply Function CRD",
			"function", row.Name, "project_id", row.ProjectID, "error", err)
	}
}

// syncBuildStatus fetches CRD status and updates meta DB if there is a diff.
// If CRD is missing, it attempts self-healing by recreating the CRD.
func (s *FunctionService) syncBuildStatus(ctx context.Context, row store.Function) store.Function {
	if s.crdMgr == nil || row.Status == "deleted" {
		return row
	}

	crdStatus, err := s.crdMgr.GetStatus(ctx, row.ProjectID, row.Name)
	if errors.Is(err, k8s.ErrCRDNotFound) {
		// Self-healing: CRD missing → recreate
		slog.WarnContext(ctx, "Function CRD not found, attempting self-healing",
			"function", row.Name, "project_id", row.ProjectID)
		s.applyCRD(ctx, row)
		return row
	}
	if err != nil {
		slog.WarnContext(ctx, "failed to get Function CRD status",
			"function", row.Name, "project_id", row.ProjectID, "error", err)
		return row
	}

	// Map CRD phase → meta DB status
	dbStatus := mapPhaseToStatus(crdStatus.Phase)
	if dbStatus == "" {
		return row // CRD has no status yet
	}

	// Check if meta DB needs an update
	if row.Status == dbStatus &&
		ptrStringEqual(row.BuildImageRef, crdStatus.ImageRef) &&
		ptrStringEqual(row.BuildImageDigest, crdStatus.ImageDigest) {
		return row // no diff
	}

	lastBuiltAt := pgtype.Timestamptz{}
	if crdStatus.LastBuiltAt != nil {
		lastBuiltAt = pgtype.Timestamptz{Time: *crdStatus.LastBuiltAt, Valid: true}
	}

	updated, err := s.q.UpdateFunctionBuildStatus(ctx, store.UpdateFunctionBuildStatusParams{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		Status:           dbStatus,
		BuildImageRef:    nilIfEmpty(crdStatus.ImageRef),
		BuildImageDigest: nilIfEmpty(crdStatus.ImageDigest),
		BuildDurationSec: crdStatus.BuildDurationSec,
		LastBuiltAt:      lastBuiltAt,
	})
	if err != nil {
		slog.WarnContext(ctx, "failed to sync build status to meta DB",
			"function", row.Name, "project_id", row.ProjectID, "error", err)
		return row
	}
	return updated
}

func mapPhaseToStatus(phase string) string {
	switch phase {
	case "Building":
		return "building"
	case "Ready":
		return "ready"
	case "Failed":
		return "failed"
	default:
		return ""
	}
}

func ptrStringEqual(ptr *string, val string) bool {
	if ptr == nil {
		return val == ""
	}
	return *ptr == val
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// validateInlineSourceSize checks that inline source code doesn't exceed 1MB.
func validateInlineSourceSize(sourceType string, sourceConfig []byte) *apperror.AppError {
	if sourceType != "inline" || len(sourceConfig) == 0 {
		return nil
	}
	var cfg InlineSourceConfig
	if err := json.Unmarshal(sourceConfig, &cfg); err != nil {
		return nil // validation of format is done elsewhere
	}
	if len(cfg.Code) > maxInlineSourceBytes {
		return apperror.New(apperror.CodeInvalidArgument, "inline source exceeds 1MB limit, use git or zip")
	}
	return nil
}
