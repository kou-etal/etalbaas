package service

import (
	"encoding/json"
	"fmt"

	"github.com/kou-etal/etalbaas/pkg/k8s"
	"github.com/kou-etal/etalbaas/services/function/internal/store"
)

// buildCRDParamsFromRow converts a meta DB Function row to FunctionCRDParams
// for creating/updating the Function CRD in Kubernetes.
func buildCRDParamsFromRow(row store.Function) (k8s.FunctionCRDParams, error) {
	params := k8s.FunctionCRDParams{
		ProjectID:   row.ProjectID,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		Kind:        row.Kind,
		SourceType:  row.SourceType,
		TimeoutSec:  row.TimeoutSec,
	}

	// Source
	switch row.SourceType {
	case "git":
		var cfg GitSourceConfig
		if err := json.Unmarshal(row.SourceConfig, &cfg); err != nil {
			return params, fmt.Errorf("unmarshal git source config: %w", err)
		}
		params.SourceGit = &k8s.GitSourceCRD{
			Repo: cfg.RepoURL,
			Ref:  cfg.Branch,
			Path: cfg.Subpath,
		}
	case "inline":
		var cfg InlineSourceConfig
		if err := json.Unmarshal(row.SourceConfig, &cfg); err != nil {
			return params, fmt.Errorf("unmarshal inline source config: %w", err)
		}
		params.SourceInline = &k8s.InlineSourceCRD{
			Files:      map[string]string{cfg.Filename: cfg.Code},
			Entrypoint: cfg.Filename,
		}
	case "zip":
		if row.SourceStoragePath == nil {
			return params, fmt.Errorf("zip source missing storage_path")
		}
		params.SourceZip = &k8s.ZipSourceCRD{
			StoragePath: *row.SourceStoragePath,
		}
	}

	// Runtime
	if row.RuntimePreset != nil {
		params.RuntimePreset = *row.RuntimePreset
	}
	if row.RuntimeDockerfile != nil {
		params.RuntimeDockerfile = *row.RuntimeDockerfile
	}
	params.RuntimeRequirements = row.RuntimeRequirements

	// GPU
	if len(row.GpuConfig) > 0 && string(row.GpuConfig) != "null" {
		var gc GpuConfigJSON
		if err := json.Unmarshal(row.GpuConfig, &gc); err != nil {
			return params, fmt.Errorf("unmarshal gpu config: %w", err)
		}
		if gc.Type != "" {
			params.GPU = &k8s.GPUConfigCRD{
				Required:       true,
				Type:           gc.Type,
				Provider:       gc.Provider,
				Product:        gc.Product,
				ProviderConfig: gc.ProviderConfig,
			}
		}
	}

	// Triggers
	if len(row.Triggers) > 0 {
		var triggers []TriggerJSON
		if err := json.Unmarshal(row.Triggers, &triggers); err != nil {
			return params, fmt.Errorf("unmarshal triggers: %w", err)
		}
		for _, t := range triggers {
			crdTrigger := convertTriggerToCRD(t)
			if crdTrigger != nil {
				params.Triggers = append(params.Triggers, *crdTrigger)
			}
		}
	}

	// Env vars
	if len(row.EnvVars) > 0 && string(row.EnvVars) != "{}" {
		var envVars []EnvVarJSON
		if err := json.Unmarshal(row.EnvVars, &envVars); err != nil {
			return params, fmt.Errorf("unmarshal env_vars: %w", err)
		}
		for _, ev := range envVars {
			params.EnvVars = append(params.EnvVars, k8s.EnvVarCRD{
				Name:       ev.Name,
				Value:      ev.Value,
				SecretName: ev.SecretName,
			})
		}
	}

	return params, nil
}

func convertTriggerToCRD(t TriggerJSON) *k8s.TriggerCRD {
	switch t.Type {
	case "database_change":
		if t.DatabaseChange == nil {
			return nil
		}
		return &k8s.TriggerCRD{
			Type: "DatabaseChange",
			DatabaseChange: &k8s.DatabaseChangeTriggerCRD{
				Table:          t.DatabaseChange.Table,
				Operations:     t.DatabaseChange.Events,
				Filter:         t.DatabaseChange.Filter,
				IncludeColumns: t.DatabaseChange.IncludeColumns,
			},
		}
	case "object_storage":
		if t.ObjectStorage == nil {
			return nil
		}
		return &k8s.TriggerCRD{
			Type: "ObjectStorage",
			ObjectStorage: &k8s.ObjectStorageTriggerCRD{
				Bucket: t.ObjectStorage.Bucket,
				Events: t.ObjectStorage.Events,
				Prefix: t.ObjectStorage.Prefix,
			},
		}
	default:
		return nil
	}
}
