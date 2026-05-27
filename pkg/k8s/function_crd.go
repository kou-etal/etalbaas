package k8s

import (
	"context"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// ErrCRDNotFound indicates the Function CRD does not exist.
var ErrCRDNotFound = errors.New("function CRD not found")

var functionGVR = schema.GroupVersionResource{
	Group:    "etalbaas.io",
	Version:  "v1alpha1",
	Resource: "functions",
}

// FunctionCRDManager manages Function CRDs using the dynamic K8s client.
type FunctionCRDManager interface {
	CreateOrUpdate(ctx context.Context, params FunctionCRDParams) error
	Delete(ctx context.Context, projectID, functionName string) error
	GetStatus(ctx context.Context, projectID, functionName string) (*FunctionCRDStatus, error)
}

// FunctionCRDParams holds the parameters needed to build a Function CRD.
type FunctionCRDParams struct {
	ProjectID   string
	Name        string
	DisplayName string
	Kind        string // "heavy-job" | "heavy-deployment" | "light-deployment"

	SourceType   string
	SourceGit    *GitSourceCRD
	SourceInline *InlineSourceCRD
	SourceZip    *ZipSourceCRD

	RuntimePreset       string
	RuntimeRequirements []string
	RuntimeDockerfile   string

	TimeoutSec int32
	GPU        *GPUConfigCRD
	Triggers   []TriggerCRD
	EnvVars    []EnvVarCRD
}

// GitSourceCRD represents a git source for a Function CRD.
type GitSourceCRD struct {
	Repo string
	Ref  string
	Path string
}

// InlineSourceCRD represents an inline source for a Function CRD.
type InlineSourceCRD struct {
	Files      map[string]string
	Entrypoint string
}

// ZipSourceCRD represents a zip source for a Function CRD.
type ZipSourceCRD struct {
	StoragePath string
	Sha256      string
}

// GPUConfigCRD represents GPU configuration for a Function CRD.
type GPUConfigCRD struct {
	Required       bool
	Type           string
	Provider       string
	Product        string
	ProviderConfig map[string]string
}

// TriggerCRD represents a trigger for a Function CRD.
type TriggerCRD struct {
	Type           string
	Http           *HttpTriggerCRD
	DatabaseChange *DatabaseChangeTriggerCRD
	ObjectStorage  *ObjectStorageTriggerCRD
}

// HttpTriggerCRD represents an HTTP trigger.
type HttpTriggerCRD struct {
	Path           string
	Authentication string
}

// DatabaseChangeTriggerCRD represents a database change trigger.
type DatabaseChangeTriggerCRD struct {
	Table          string
	Operations     []string
	Filter         string
	IncludeColumns []string
}

// ObjectStorageTriggerCRD represents an object storage trigger.
type ObjectStorageTriggerCRD struct {
	Bucket string
	Prefix string
	Events []string
}

// EnvVarCRD represents an environment variable for a Function CRD.
type EnvVarCRD struct {
	Name       string
	Value      string
	SecretName string
}

// FunctionCRDStatus holds the parsed status from a Function CRD.
type FunctionCRDStatus struct {
	Phase            string
	ImageRef         string
	ImageDigest      string
	BuildDurationSec *int32
	LastBuiltAt      *time.Time
}

type functionCRDManager struct {
	client dynamic.Interface
}

// NewFunctionCRDManager creates a new FunctionCRDManager.
func NewFunctionCRDManager(client dynamic.Interface) FunctionCRDManager {
	return &functionCRDManager{client: client}
}

func (m *functionCRDManager) CreateOrUpdate(ctx context.Context, params FunctionCRDParams) error {
	namespace := "project-" + params.ProjectID
	resource := m.client.Resource(functionGVR).Namespace(namespace)

	obj := buildFunctionCRDObject(params)

	existing, err := resource.Get(ctx, params.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = resource.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Function CRD: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Function CRD: %w", err)
	}

	// Preserve resourceVersion for update.
	obj.SetResourceVersion(existing.GetResourceVersion())
	_, err = resource.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Function CRD: %w", err)
	}
	return nil
}

func (m *functionCRDManager) Delete(ctx context.Context, projectID, functionName string) error {
	namespace := "project-" + projectID
	err := m.client.Resource(functionGVR).Namespace(namespace).Delete(ctx, functionName, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil // already gone
	}
	if err != nil {
		return fmt.Errorf("delete Function CRD: %w", err)
	}
	return nil
}

func (m *functionCRDManager) GetStatus(ctx context.Context, projectID, functionName string) (*FunctionCRDStatus, error) {
	namespace := "project-" + projectID
	obj, err := m.client.Resource(functionGVR).Namespace(namespace).Get(ctx, functionName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, ErrCRDNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get Function CRD: %w", err)
	}
	return parseStatus(obj)
}

// buildFunctionCRDObject constructs the unstructured Function CRD from params.
// JSON field names match operator/api/v1alpha1/function_types.go tags.
func buildFunctionCRDObject(params FunctionCRDParams) *unstructured.Unstructured {
	namespace := "project-" + params.ProjectID

	source := map[string]interface{}{
		"type": params.SourceType,
	}
	switch params.SourceType {
	case "git":
		if params.SourceGit != nil {
			git := map[string]interface{}{
				"repo": params.SourceGit.Repo,
			}
			if params.SourceGit.Ref != "" {
				git["ref"] = params.SourceGit.Ref
			}
			if params.SourceGit.Path != "" {
				git["path"] = params.SourceGit.Path
			}
			source["git"] = git
		}
	case "inline":
		if params.SourceInline != nil {
			source["inline"] = map[string]interface{}{
				"files":      toUnstructuredStringMap(params.SourceInline.Files),
				"entrypoint": params.SourceInline.Entrypoint,
			}
		}
	case "zip":
		if params.SourceZip != nil {
			zip := map[string]interface{}{
				"storagePath": params.SourceZip.StoragePath,
			}
			if params.SourceZip.Sha256 != "" {
				zip["sha256"] = params.SourceZip.Sha256
			}
			source["zip"] = zip
		}
	}

	runtime := map[string]interface{}{
		"preset": params.RuntimePreset,
	}
	if params.RuntimeDockerfile != "" {
		runtime["preset"] = "custom"
		runtime["dockerfile"] = params.RuntimeDockerfile
	}
	if len(params.RuntimeRequirements) > 0 {
		runtime["requirements"] = toInterfaceSlice(params.RuntimeRequirements)
	}

	spec := map[string]interface{}{
		"displayName": params.DisplayName,
		"projectRef":  map[string]interface{}{"name": params.ProjectID},
		"kind":        params.Kind,
		"source":      source,
		"runtime":     runtime,
	}

	if params.GPU != nil && params.GPU.Required {
		gpuSpec := map[string]interface{}{
			"required": true,
			"type":     params.GPU.Type,
			"provider": params.GPU.Provider,
			"product":  params.GPU.Product,
		}
		if len(params.GPU.ProviderConfig) > 0 {
			pc := make(map[string]interface{}, len(params.GPU.ProviderConfig))
			for k, v := range params.GPU.ProviderConfig {
				pc[k] = v
			}
			gpuSpec["providerConfig"] = pc
		}
		spec["gpu"] = gpuSpec
	}

	if len(params.Triggers) > 0 {
		triggers := make([]interface{}, 0, len(params.Triggers))
		for _, t := range params.Triggers {
			triggers = append(triggers, buildTriggerSpec(t))
		}
		spec["triggers"] = triggers
	}

	if len(params.EnvVars) > 0 {
		envVars := make([]interface{}, 0, len(params.EnvVars))
		for _, e := range params.EnvVars {
			ev := map[string]interface{}{
				"name": e.Name,
			}
			if e.Value != "" {
				ev["value"] = e.Value
			}
			if e.SecretName != "" {
				ev["secretName"] = e.SecretName
			}
			envVars = append(envVars, ev)
		}
		spec["env"] = envVars
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "etalbaas.io/v1alpha1",
			"kind":       "Function",
			"metadata": map[string]interface{}{
				"name":      params.Name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"etalbaas.io/project-id": params.ProjectID,
				},
			},
			"spec": spec,
		},
	}
}

func buildTriggerSpec(t TriggerCRD) map[string]interface{} {
	trigger := map[string]interface{}{
		"type": t.Type,
	}
	if t.Http != nil {
		http := map[string]interface{}{}
		if t.Http.Path != "" {
			http["path"] = t.Http.Path
		}
		if t.Http.Authentication != "" {
			http["authentication"] = t.Http.Authentication
		}
		trigger["http"] = http
	}
	if t.DatabaseChange != nil {
		dc := map[string]interface{}{
			"table":      t.DatabaseChange.Table,
			"operations": toInterfaceSlice(t.DatabaseChange.Operations),
		}
		if t.DatabaseChange.Filter != "" {
			dc["filter"] = t.DatabaseChange.Filter
		}
		if len(t.DatabaseChange.IncludeColumns) > 0 {
			dc["includeColumns"] = toInterfaceSlice(t.DatabaseChange.IncludeColumns)
		}
		trigger["databaseChange"] = dc
	}
	if t.ObjectStorage != nil {
		os := map[string]interface{}{
			"bucket": t.ObjectStorage.Bucket,
			"events": toInterfaceSlice(t.ObjectStorage.Events),
		}
		if t.ObjectStorage.Prefix != "" {
			os["prefix"] = t.ObjectStorage.Prefix
		}
		trigger["objectStorage"] = os
	}
	return trigger
}

// parseStatus extracts FunctionCRDStatus from an unstructured Function CRD.
func parseStatus(obj *unstructured.Unstructured) (*FunctionCRDStatus, error) {
	status, ok, _ := unstructured.NestedMap(obj.Object, "status")
	if !ok {
		return &FunctionCRDStatus{}, nil
	}

	result := &FunctionCRDStatus{}

	if phase, ok, _ := unstructured.NestedString(status, "phase"); ok {
		result.Phase = phase
	}

	if build, ok, _ := unstructured.NestedMap(status, "build"); ok {
		if ref, ok, _ := unstructured.NestedString(build, "imageRef"); ok {
			result.ImageRef = ref
		}
		if digest, ok, _ := unstructured.NestedString(build, "imageDigest"); ok {
			result.ImageDigest = digest
		}
		if dur, ok, _ := unstructured.NestedInt64(build, "buildDurationSeconds"); ok {
			d := int32(dur)
			result.BuildDurationSec = &d
		}
		if builtAt, ok, _ := unstructured.NestedString(build, "lastBuiltAt"); ok && builtAt != "" {
			if t, err := time.Parse(time.RFC3339, builtAt); err == nil {
				result.LastBuiltAt = &t
			}
		}
	}

	return result, nil
}

func toUnstructuredStringMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func toInterfaceSlice(ss []string) []interface{} {
	result := make([]interface{}, len(ss))
	for i, s := range ss {
		result[i] = s
	}
	return result
}
