package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	commonv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/common/v1"
	functionv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1"
	"github.com/kou-etal/etalbaas/services/function/internal/service"
	"github.com/kou-etal/etalbaas/services/function/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxTokenLen = 256

// ============================================================
// Proto → DB (write path)
// ============================================================

// SourceFromProto converts proto source oneof into DB columns.
// Returns (sourceType, sourceConfig JSON, sourceStoragePath).
func SourceFromProto(git *functionv1.GitSource, zip *functionv1.ZipSource, inline *functionv1.InlineSource) (string, []byte, *string, error) {
	switch {
	case git != nil:
		cfg, err := json.Marshal(service.GitSourceConfig{
			RepoURL: git.RepoUrl,
			Branch:  git.Branch,
			Subpath: git.Subpath,
		})
		if err != nil {
			return "", nil, nil, fmt.Errorf("marshal git config: %w", err)
		}
		return "git", cfg, nil, nil

	case zip != nil:
		path := zip.StoragePath
		return "zip", []byte("{}"), &path, nil

	case inline != nil:
		cfg, err := json.Marshal(service.InlineSourceConfig{
			Code:     inline.Code,
			Filename: inline.Filename,
		})
		if err != nil {
			return "", nil, nil, fmt.Errorf("marshal inline config: %w", err)
		}
		return "inline", cfg, nil, nil

	default:
		return "", nil, nil, fmt.Errorf("source must be specified")
	}
}

// RuntimeFromProto converts proto runtime oneof into DB columns.
func RuntimeFromProto(preset *functionv1.PresetRuntime, custom *functionv1.CustomRuntime) (*string, []string, *string) {
	switch {
	case preset != nil:
		p := preset.Preset
		return &p, preset.Requirements, nil
	case custom != nil:
		d := custom.Dockerfile
		return nil, nil, &d
	default:
		return nil, nil, nil
	}
}

// GpuConfigFromProto converts proto GpuConfig to JSONB bytes.
func GpuConfigFromProto(gc *functionv1.GpuConfig) []byte {
	if gc == nil {
		return nil
	}
	b, _ := json.Marshal(service.GpuConfigJSON{
		Type:     gc.Type,
		Provider: gc.Provider,
		Product:  gc.Product,
	})
	return b
}

// TriggersFromProto converts proto Trigger slice to JSONB bytes.
func TriggersFromProto(triggers []*functionv1.Trigger) ([]byte, error) {
	if len(triggers) == 0 {
		return []byte("[]"), nil
	}
	result := make([]service.TriggerJSON, len(triggers))
	for i, t := range triggers {
		switch cfg := t.Config.(type) {
		case *functionv1.Trigger_DatabaseChange:
			dc := cfg.DatabaseChange
			result[i] = service.TriggerJSON{
				Type: "database_change",
				DatabaseChange: &service.DatabaseChangeTriggerJSON{
					Table:          dc.Table,
					Events:         dc.Events,
					Filter:         dc.GetFilter(),
					IncludeColumns: dc.IncludeColumns,
				},
			}
		case *functionv1.Trigger_ObjectStorage:
			os := cfg.ObjectStorage
			result[i] = service.TriggerJSON{
				Type: "object_storage",
				ObjectStorage: &service.ObjectStorageTriggerJSON{
					Bucket: os.Bucket,
					Events: os.Events,
					Prefix: os.Prefix,
				},
			}
		default:
			return nil, fmt.Errorf("unknown trigger config type")
		}
	}
	return json.Marshal(result)
}

// EnvVarsFromProto converts proto EnvVar slice to JSONB bytes.
func EnvVarsFromProto(envVars []*functionv1.EnvVar) ([]byte, error) {
	if len(envVars) == 0 {
		return []byte("[]"), nil
	}
	result := make([]service.EnvVarJSON, len(envVars))
	for i, ev := range envVars {
		j := service.EnvVarJSON{Name: ev.Name}
		switch src := ev.Source.(type) {
		case *functionv1.EnvVar_Value:
			j.Value = src.Value
		case *functionv1.EnvVar_SecretName:
			j.SecretName = src.SecretName
		}
		result[i] = j
	}
	return json.Marshal(result)
}

// ============================================================
// DB → Proto (read path)
// ============================================================

// FunctionToProto converts a DB Function row to a proto Function message.
func FunctionToProto(row store.Function) (*functionv1.Function, error) {
	f := &functionv1.Function{
		Id:          row.ID.String(),
		ProjectId:   row.ProjectID,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		Kind:        row.Kind,
		Mode:        row.Mode,
		TimeoutSec:  row.TimeoutSec,
		Status:      row.Status,
		CreatedAt:   timestamppb.New(row.CreatedAt),
		UpdatedAt:   timestamppb.New(row.UpdatedAt),
	}

	// Build output fields
	if row.BuildImageRef != nil {
		f.BuildImageRef = *row.BuildImageRef
	}
	if row.BuildImageDigest != nil {
		f.BuildImageDigest = *row.BuildImageDigest
	}
	if row.BuildDurationSec != nil {
		f.BuildDurationSec = *row.BuildDurationSec
	}
	if row.LastBuiltAt.Valid {
		f.LastBuiltAt = timestamppb.New(row.LastBuiltAt.Time)
	}

	// Source
	switch row.SourceType {
	case "git":
		var cfg service.GitSourceConfig
		if err := json.Unmarshal(row.SourceConfig, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal git config: %w", err)
		}
		f.Source = &functionv1.Function_GitSource{
			GitSource: &functionv1.GitSource{
				RepoUrl: cfg.RepoURL,
				Branch:  cfg.Branch,
				Subpath: cfg.Subpath,
			},
		}
	case "zip":
		var path string
		if row.SourceStoragePath != nil {
			path = *row.SourceStoragePath
		}
		f.Source = &functionv1.Function_ZipSource{
			ZipSource: &functionv1.ZipSource{StoragePath: path},
		}
	case "inline":
		var cfg service.InlineSourceConfig
		if err := json.Unmarshal(row.SourceConfig, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal inline config: %w", err)
		}
		f.Source = &functionv1.Function_InlineSource{
			InlineSource: &functionv1.InlineSource{
				Code:     cfg.Code,
				Filename: cfg.Filename,
			},
		}
	}

	// Runtime
	if row.RuntimePreset != nil {
		f.Runtime = &functionv1.Function_PresetRuntime{
			PresetRuntime: &functionv1.PresetRuntime{
				Preset:       *row.RuntimePreset,
				Requirements: row.RuntimeRequirements,
			},
		}
	} else if row.RuntimeDockerfile != nil {
		f.Runtime = &functionv1.Function_CustomRuntime{
			CustomRuntime: &functionv1.CustomRuntime{
				Dockerfile: *row.RuntimeDockerfile,
			},
		}
	}

	// GPU config
	if len(row.GpuConfig) > 0 && string(row.GpuConfig) != "null" {
		var gc service.GpuConfigJSON
		if err := json.Unmarshal(row.GpuConfig, &gc); err != nil {
			return nil, fmt.Errorf("unmarshal gpu config: %w", err)
		}
		f.GpuConfig = &functionv1.GpuConfig{
			Type:     gc.Type,
			Provider: gc.Provider,
			Product:  gc.Product,
		}
	}

	// Triggers
	if len(row.Triggers) > 0 {
		var triggers []service.TriggerJSON
		if err := json.Unmarshal(row.Triggers, &triggers); err != nil {
			return nil, fmt.Errorf("unmarshal triggers: %w", err)
		}
		for _, tj := range triggers {
			switch tj.Type {
			case "database_change":
				if tj.DatabaseChange == nil {
					return nil, fmt.Errorf("trigger type database_change with nil config")
				}
				dc := tj.DatabaseChange
				t := &functionv1.Trigger{
					Config: &functionv1.Trigger_DatabaseChange{
						DatabaseChange: &functionv1.DatabaseChangeTrigger{
							Table:          dc.Table,
							Events:         dc.Events,
							IncludeColumns: dc.IncludeColumns,
						},
					},
				}
				if dc.Filter != "" {
					t.Config.(*functionv1.Trigger_DatabaseChange).DatabaseChange.Filter = &dc.Filter
				}
				f.Triggers = append(f.Triggers, t)
			case "object_storage":
				if tj.ObjectStorage == nil {
					return nil, fmt.Errorf("trigger type object_storage with nil config")
				}
				os := tj.ObjectStorage
				f.Triggers = append(f.Triggers, &functionv1.Trigger{
					Config: &functionv1.Trigger_ObjectStorage{
						ObjectStorage: &functionv1.ObjectStorageTrigger{
							Bucket: os.Bucket,
							Events: os.Events,
							Prefix: os.Prefix,
						},
					},
				})
			default:
				return nil, fmt.Errorf("unknown trigger type: %s", tj.Type)
			}
		}
	}

	// Env vars — handle DB default '{}' (object) gracefully alongside expected '[]' (array)
	if len(row.EnvVars) > 0 && string(row.EnvVars) != "{}" {
		var envVars []service.EnvVarJSON
		if err := json.Unmarshal(row.EnvVars, &envVars); err != nil {
			return nil, fmt.Errorf("unmarshal env_vars: %w", err)
		}
		f.EnvVars = make([]*functionv1.EnvVar, len(envVars))
		for i, ev := range envVars {
			pev := &functionv1.EnvVar{Name: ev.Name}
			if ev.SecretName != "" {
				pev.Source = &functionv1.EnvVar_SecretName{SecretName: ev.SecretName}
			} else {
				pev.Source = &functionv1.EnvVar_Value{Value: ev.Value}
			}
			f.EnvVars[i] = pev
		}
	}

	return f, nil
}

// FunctionsToProto converts a slice of DB rows to proto messages with pagination cursor.
func FunctionsToProto(rows []store.Function, hasMore bool) ([]*functionv1.Function, string, error) {
	functions := make([]*functionv1.Function, len(rows))
	for i, r := range rows {
		f, err := FunctionToProto(r)
		if err != nil {
			return nil, "", err
		}
		functions[i] = f
	}
	var nextToken string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = encodeUUIDCursor(last.CreatedAt, last.ID)
	}
	return functions, nextToken, nil
}

// InvocationToProto converts a DB Invocation row to a proto Invocation message.
func InvocationToProto(row store.Invocation) *functionv1.Invocation {
	inv := &functionv1.Invocation{
		Id:          row.ID.String(),
		FunctionId:  row.FunctionID.String(),
		ProjectId:   row.ProjectID,
		TriggerType: row.TriggerType,
		Mode:        row.Mode,
		Status:      row.Status,
		RetryCount:  row.RetryCount,
		CreatedAt:   timestamppb.New(row.CreatedAt),
	}

	if row.ErrorMessage != nil {
		inv.ErrorMessage = *row.ErrorMessage
	}
	if row.DurationMs != nil {
		inv.DurationMs = *row.DurationMs
	}
	if row.ColdStartMs != nil {
		inv.ColdStartMs = *row.ColdStartMs
	}
	if row.GpuDurationMs != nil {
		inv.GpuDurationMs = *row.GpuDurationMs
	}
	if row.MemoryPeakBytes != nil {
		inv.MemoryPeakBytes = *row.MemoryPeakBytes
	}
	if row.CpuMillis != nil {
		inv.CpuMillis = *row.CpuMillis
	}
	if row.GpuProvider != nil {
		inv.GpuProvider = *row.GpuProvider
	}
	if row.GpuType != nil {
		inv.GpuType = *row.GpuType
	}
	if row.TraceID != nil {
		inv.TraceId = *row.TraceID
	}
	if row.StartedAt.Valid {
		inv.StartedAt = timestamppb.New(row.StartedAt.Time)
	}
	if row.CompletedAt.Valid {
		inv.CompletedAt = timestamppb.New(row.CompletedAt.Time)
	}
	return inv
}

// InvocationsToProto converts a slice of DB invocation rows to proto messages with pagination cursor.
func InvocationsToProto(rows []store.Invocation, hasMore bool) ([]*functionv1.Invocation, string) {
	invocations := make([]*functionv1.Invocation, len(rows))
	for i, r := range rows {
		invocations[i] = InvocationToProto(r)
	}
	var nextToken string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = encodeUUIDCursor(last.CreatedAt, last.ID)
	}
	return invocations, nextToken
}

// ============================================================
// Pagination
// ============================================================

func UUIDPaginationFromProto(p *commonv1.PaginationRequest) (int32, *time.Time, *uuid.UUID, error) {
	pageSize := int32(20)
	if p != nil && p.PageSize != 0 {
		if p.PageSize < 0 {
			return 0, nil, nil, apperror.New(apperror.CodeInvalidArgument, "page_size must be positive")
		}
		pageSize = p.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if p != nil && p.PageToken != "" {
		t, id, err := decodeUUIDCursor(p.PageToken)
		if err != nil {
			return 0, nil, nil, apperror.New(apperror.CodeInvalidArgument, "invalid page_token")
		}
		return pageSize, &t, &id, nil
	}
	return pageSize, nil, nil, nil
}

func PaginationToProto(nextToken string) *commonv1.PaginationResponse {
	if nextToken == "" {
		return nil
	}
	return &commonv1.PaginationResponse{
		NextPageToken: nextToken,
	}
}

// --- UUID cursor (time + uuid.UUID) ---

func encodeUUIDCursor(t time.Time, id uuid.UUID) string {
	raw := t.Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeUUIDCursor(s string) (time.Time, uuid.UUID, error) {
	if len(s) > maxTokenLen {
		return time.Time{}, uuid.Nil, fmt.Errorf("token too long")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("decode base64: %w", err)
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor format")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("parse time: %w", err)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}
	return t, id, nil
}

// --- Test helpers ---

func EncodeUUIDCursorForTest(t time.Time, id uuid.UUID) string {
	return encodeUUIDCursor(t, id)
}
