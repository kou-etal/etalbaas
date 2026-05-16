package resources

import (
	"fmt"

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
				"instances":             replicas,
				"imageName":             "ghcr.io/cloudnative-pg/postgresql:" + version,
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
						"database":    "postgres",
						"owner":       "app",
						"postInitSQL": buildPostInitSQL(pg.Extensions),
						"postInitApplicationSQL": []interface{}{
							fmt.Sprintf("CREATE PUBLICATION cdc_%s FOR ALL TABLES", projectID),
						},
					},
				},
				// CDC dedicated user: REPLICATION + pg_read_all_data (SELECT on all tables).
				// Avoids using superuser for CDC Pod — limits blast radius.
				"managed": map[string]interface{}{
					"roles": []interface{}{
						map[string]interface{}{
							"name":        "cdc",
							"ensure":      "present",
							"login":       true,
							"replication": true,
							"inRoles":     []interface{}{"pg_read_all_data"},
							"passwordSecret": map[string]interface{}{
								"name": "db-cdc",
							},
						},
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
	stmts := make([]interface{}, 0, len(extensions)+30)
	for _, ext := range extensions {
		stmts = append(stmts, "CREATE EXTENSION IF NOT EXISTS "+ext+";")
	}

	// Auth schema: roles + helper functions (Supabase-compatible).
	// Required for PostgREST SET LOCAL ROLE and Storage MS RLS enforcement.
	stmts = append(stmts,
		"CREATE ROLE anon NOLOGIN;",
		"CREATE ROLE authenticated NOLOGIN;",
		"CREATE ROLE service_role NOLOGIN;",
		"GRANT anon, authenticated, service_role TO app;",
		"CREATE SCHEMA IF NOT EXISTS auth;",
		`CREATE OR REPLACE FUNCTION auth.uid() RETURNS UUID LANGUAGE sql STABLE AS $$ SELECT NULLIF(current_setting('request.jwt.claims', true)::json->>'sub', '')::UUID $$;`,
		`CREATE OR REPLACE FUNCTION auth.role() RETURNS TEXT LANGUAGE sql STABLE AS $$ SELECT NULLIF(current_setting('request.jwt.claims', true)::json->>'role', '') $$;`,
		"GRANT USAGE ON SCHEMA auth TO anon, authenticated, service_role;",
		"GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO anon, authenticated, service_role;",
	)

	// Storage schema: bucket/object metadata tables with RLS.
	stmts = append(stmts,
		"CREATE SCHEMA IF NOT EXISTS storage;",
		"GRANT USAGE ON SCHEMA storage TO anon, authenticated, service_role;",
		`CREATE TABLE IF NOT EXISTS storage.buckets (id TEXT PRIMARY KEY, name TEXT NOT NULL, project_id TEXT NOT NULL, access_level TEXT NOT NULL DEFAULT 'protected', file_size_limit BIGINT, allowed_mime_types TEXT[], created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(project_id, name));`,
		`CREATE TABLE IF NOT EXISTS storage.objects (id UUID DEFAULT gen_random_uuid() PRIMARY KEY, bucket_id TEXT NOT NULL REFERENCES storage.buckets(id) ON DELETE CASCADE, name TEXT NOT NULL, owner UUID, size BIGINT, mime_type TEXT, etag TEXT, metadata JSONB DEFAULT '{}', created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(bucket_id, name));`,
		"ALTER TABLE storage.buckets ENABLE ROW LEVEL SECURITY;",
		"ALTER TABLE storage.objects ENABLE ROW LEVEL SECURITY;",
		"GRANT SELECT ON ALL TABLES IN SCHEMA storage TO anon, authenticated;",
		"GRANT INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA storage TO authenticated;",
		"GRANT ALL ON ALL TABLES IN SCHEMA storage TO service_role;",
		`CREATE POLICY "authenticated users can CRUD own objects" ON storage.objects FOR ALL USING (owner = auth.uid()) WITH CHECK (owner = auth.uid());`,
		`CREATE POLICY "anon can read public bucket objects" ON storage.objects FOR SELECT USING (EXISTS (SELECT 1 FROM storage.buckets b WHERE b.id = bucket_id AND b.access_level = 'public'));`,
		`CREATE POLICY "service_role bypass objects" ON storage.objects FOR ALL USING (auth.role() = 'service_role');`,
		`CREATE POLICY "service_role bypass buckets" ON storage.buckets FOR ALL USING (auth.role() = 'service_role');`,
	)

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
