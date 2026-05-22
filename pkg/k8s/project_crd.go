package k8s

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var projectGVR = schema.GroupVersionResource{
	Group:    "etalbaas.io",
	Version:  "v1alpha1",
	Resource: "projects",
}

// ProjectCRDManager manages Project CRDs using the dynamic K8s client.
type ProjectCRDManager interface {
	CreateOrUpdate(ctx context.Context, params ProjectCRDParams) error
	Delete(ctx context.Context, projectID string) error
}

// ProjectCRDParams holds the parameters needed to build a Project CRD.
type ProjectCRDParams struct {
	ProjectID          string
	DisplayName        string
	Description        string
	PostgresEnabled    bool
	PostgresExtensions []string
	RedisEnabled       bool
	PostgrestEnabled   bool
}

type projectCRDManager struct {
	client    dynamic.Interface
	namespace string
}

// NewProjectCRDManager creates a new ProjectCRDManager.
func NewProjectCRDManager(client dynamic.Interface, namespace string) ProjectCRDManager {
	return &projectCRDManager{client: client, namespace: namespace}
}

func (m *projectCRDManager) CreateOrUpdate(ctx context.Context, params ProjectCRDParams) error {
	resource := m.client.Resource(projectGVR).Namespace(m.namespace)

	obj := buildProjectCRDObject(params, m.namespace)

	existing, err := resource.Get(ctx, params.ProjectID, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = resource.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Project CRD: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Project CRD: %w", err)
	}

	// Preserve resourceVersion for update.
	obj.SetResourceVersion(existing.GetResourceVersion())
	_, err = resource.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Project CRD: %w", err)
	}
	return nil
}

func (m *projectCRDManager) Delete(ctx context.Context, projectID string) error {
	err := m.client.Resource(projectGVR).Namespace(m.namespace).Delete(ctx, projectID, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil // already gone
	}
	if err != nil {
		return fmt.Errorf("delete Project CRD: %w", err)
	}
	return nil
}

// buildProjectCRDObject constructs the unstructured Project CRD from params.
// JSON field names match operator/api/v1alpha1/project_types.go tags.
func buildProjectCRDObject(params ProjectCRDParams, namespace string) *unstructured.Unstructured {
	stack := map[string]interface{}{}

	if params.PostgresEnabled {
		pg := map[string]interface{}{
			"enabled": true,
			"version": "16",
		}
		if len(params.PostgresExtensions) > 0 {
			pg["extensions"] = toInterfaceSlice(params.PostgresExtensions)
		}
		stack["postgres"] = pg
	}

	if params.RedisEnabled {
		stack["redis"] = map[string]interface{}{
			"enabled": true,
		}
	}

	if params.PostgrestEnabled {
		stack["postgrest"] = map[string]interface{}{
			"enabled": true,
		}
	}

	spec := map[string]interface{}{
		"displayName": params.DisplayName,
		"stack":       stack,
		"networking": map[string]interface{}{
			"subdomain": params.ProjectID,
		},
		"plan": "free",
	}

	if params.Description != "" {
		spec["description"] = params.Description
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "etalbaas.io/v1alpha1",
			"kind":       "Project",
			"metadata": map[string]interface{}{
				"name":      params.ProjectID,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"etalbaas.io/project-id": params.ProjectID,
				},
			},
			"spec": spec,
		},
	}
}
