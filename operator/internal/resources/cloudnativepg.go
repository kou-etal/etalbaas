package resources

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
)

// ClusterGVK returns the GroupVersionKind for CloudNativePG Cluster.
func ClusterGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "postgresql.cnpg.io",
		Version: "v1",
		Kind:    cnpgClusterKind,
	}
}

// PoolerGVK returns the GroupVersionKind for CloudNativePG Pooler.
func PoolerGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "postgresql.cnpg.io",
		Version: "v1",
		Kind:    cnpgPoolerKind,
	}
}

const (
	cnpgAPIVersion = "postgresql.cnpg.io/v1"
	cnpgClusterKind = "Cluster"
	cnpgPoolerKind  = "Pooler"
)

// DesiredCloudNativePGCluster builds the desired CloudNativePG Cluster CR.
func DesiredCloudNativePGCluster(project *etalbaasv1alpha1.Project) *unstructured.Unstructured {
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	replicas := int64(2)
	if pg.Replicas != nil {
		replicas = int64(*pg.Replicas)
	}

	version := "16"
	if pg.Version != "" {
		version = pg.Version
	}

	storage := "10Gi"
	if pg.Storage != "" {
		storage = pg.Storage
	}

	cpuReq := "500m"
	memReq := "1Gi"
	if pg.Resources != nil {
		if pg.Resources.CPU != "" {
			cpuReq = pg.Resources.CPU
		}
		if pg.Resources.Memory != "" {
			memReq = pg.Resources.Memory
		}
	}

	// Build PostgreSQL parameters
	// NOTE: sslnegotiation=direct is a libpq client param, not a server GUC.
	// It is set in the connection URI, not in postgresql.conf.
	pgParameters := map[string]interface{}{
		"shared_buffers":        "256MB",
		"max_connections":       "100",
		"wal_level":             "logical",
		"max_wal_senders":       "10",
		"max_replication_slots": "10",
	}

	// Build shared_preload_libraries list for extensions that need it.
	// CNPG supports shared_preload_libraries as a dedicated field in spec.postgresql.
	var sharedPreloadLibraries []interface{}
	if containsString(pg.Extensions, "pgvector") {
		sharedPreloadLibraries = append(sharedPreloadLibraries, "vector")
	}

	cluster := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": cnpgAPIVersion,
			"kind":       cnpgClusterKind,
			"metadata": map[string]interface{}{
				"name":      "db",
				"namespace": namespace,
				"labels":    toUnstructuredLabels(ComponentLabels(projectID, userID, plan, "postgres")),
			},
			"spec": map[string]interface{}{
				"instances":    replicas,
				"imageName":    "ghcr.io/cloudnative-pg/postgresql:" + version,
				"primaryUpdateStrategy": "unsupervised",
				"storage": map[string]interface{}{
					"size": storage,
				},
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    cpuReq,
						"memory": memReq,
					},
					"limits": map[string]interface{}{
						"cpu":    cpuReq,
						"memory": memReq,
					},
				},
				"postgresql": buildPostgresqlSection(pgParameters, sharedPreloadLibraries),
				"bootstrap": map[string]interface{}{
					"initdb": map[string]interface{}{
						"database": "postgres",
						"owner":    "app",
						"postInitSQL": buildPostInitSQL(pg.Extensions),
					},
				},
			},
		},
	}

	return cluster
}

// DesiredCloudNativePGPooler builds the desired CloudNativePG Pooler CR (PgBouncer).
func DesiredCloudNativePGPooler(project *etalbaasv1alpha1.Project) *unstructured.Unstructured {
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled || pg.Pooler == nil || !pg.Pooler.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	instances := int64(2)
	if pg.Pooler.Instances != nil {
		instances = int64(*pg.Pooler.Instances)
	}

	pooler := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": cnpgAPIVersion,
			"kind":       cnpgPoolerKind,
			"metadata": map[string]interface{}{
				"name":      "db-pooler",
				"namespace": namespace,
				"labels":    toUnstructuredLabels(ComponentLabels(projectID, userID, plan, "pooler")),
			},
			"spec": map[string]interface{}{
				"cluster": map[string]interface{}{
					"name": "db",
				},
				"instances": instances,
				"type":      "rw",
				"pgbouncer": map[string]interface{}{
					"poolMode": "transaction",
				},
			},
		},
	}

	return pooler
}

func buildPostgresqlSection(params map[string]interface{}, sharedPreloadLibraries []interface{}) map[string]interface{} {
	section := map[string]interface{}{
		"parameters": params,
	}
	if len(sharedPreloadLibraries) > 0 {
		section["shared_preload_libraries"] = sharedPreloadLibraries
	}
	return section
}

func buildPostInitSQL(extensions []string) []interface{} {
	stmts := make([]interface{}, 0, len(extensions))
	for _, ext := range extensions {
		stmts = append(stmts, "CREATE EXTENSION IF NOT EXISTS "+ext+";")
	}
	return stmts
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func toUnstructuredLabels(labels map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(labels))
	for k, v := range labels {
		result[k] = v
	}
	return result
}
