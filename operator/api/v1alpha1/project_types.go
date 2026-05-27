package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of a Project.
type ProjectSpec struct {
	// DisplayName is the human-readable name of the project.
	// +kubebuilder:validation:MaxLength=100
	DisplayName string `json:"displayName"`

	// Description is an optional description of the project.
	// +optional
	Description string `json:"description,omitempty"`

	// Stack defines the infrastructure components to provision.
	Stack ProjectStack `json:"stack"`

	// Networking defines the networking configuration.
	Networking NetworkingSpec `json:"networking"`

	// Plan is the billing plan for this project.
	// +kubebuilder:validation:Enum=free;pro;enterprise
	// +kubebuilder:default=free
	Plan string `json:"plan"`

	// Lifecycle defines the project lifecycle mode.
	// +optional
	Lifecycle *LifecycleSpec `json:"lifecycle,omitempty"`
}

// ProjectStack defines which infrastructure components are enabled.
type ProjectStack struct {
	// Postgres defines the PostgreSQL configuration.
	// +optional
	Postgres *PostgresSpec `json:"postgres,omitempty"`

	// Redis defines the Redis configuration.
	// +optional
	Redis *RedisSpec `json:"redis,omitempty"`

	// PostgREST defines the PostgREST configuration.
	// +optional
	PostgREST *PostgRESTSpec `json:"postgrest,omitempty"`
}

// PostgresSpec defines the PostgreSQL configuration.
type PostgresSpec struct {
	// Enabled indicates whether PostgreSQL is provisioned.
	Enabled bool `json:"enabled"`

	// Version is the PostgreSQL major version.
	// +kubebuilder:default="16"
	// +optional
	Version string `json:"version,omitempty"`

	// Extensions is a list of PostgreSQL extensions to enable.
	// +optional
	Extensions []string `json:"extensions,omitempty"`

	// Storage is the storage size for the PostgreSQL cluster.
	// +kubebuilder:default="10Gi"
	// +optional
	Storage string `json:"storage,omitempty"`

	// Replicas is the number of PostgreSQL instances (Primary + Replicas).
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources defines the CPU and memory resources for PostgreSQL.
	// +optional
	Resources *ComponentResources `json:"resources,omitempty"`

	// Pooler defines the connection pooler configuration.
	// +optional
	Pooler *PoolerSpec `json:"pooler,omitempty"`
}

// ComponentResources defines CPU and memory resources.
type ComponentResources struct {
	// CPU is the CPU resource (e.g., "500m").
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory is the memory resource (e.g., "1Gi").
	// +optional
	Memory string `json:"memory,omitempty"`
}

// PoolerSpec defines the connection pooler configuration.
type PoolerSpec struct {
	// Enabled indicates whether the connection pooler is provisioned.
	Enabled bool `json:"enabled"`

	// Instances is the number of pooler instances.
	// +kubebuilder:default=2
	// +optional
	Instances *int32 `json:"instances,omitempty"`
}

// RedisSpec defines the Redis configuration.
type RedisSpec struct {
	// Enabled indicates whether Redis is provisioned.
	Enabled bool `json:"enabled"`

	// Persistence indicates whether Redis data is persisted.
	// +optional
	Persistence bool `json:"persistence,omitempty"`
}

// PostgRESTSpec defines the PostgREST configuration.
type PostgRESTSpec struct {
	// Enabled indicates whether PostgREST is provisioned.
	Enabled bool `json:"enabled"`

	// AnonRole is the PostgreSQL role used for anonymous access.
	// +kubebuilder:default="anon"
	// +optional
	AnonRole string `json:"anonRole,omitempty"`
}

// NetworkingSpec defines the networking configuration.
type NetworkingSpec struct {
	// Subdomain is the subdomain for the API endpoint (e.g., "xk7a9bc2").
	Subdomain string `json:"subdomain"`

	// DbSubdomain is the subdomain for the database endpoint.
	// +optional
	DbSubdomain string `json:"dbSubdomain,omitempty"`
}

// LifecycleSpec defines the project lifecycle mode.
type LifecycleSpec struct {
	// Mode is the lifecycle mode.
	// +kubebuilder:validation:Enum=persistent;ephemeral;evolutionary
	// +kubebuilder:default=persistent
	Mode string `json:"mode"`
}

// ProjectStatus defines the observed state of a Project.
type ProjectStatus struct {
	// Phase is the current phase of the project.
	// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;Failed
	Phase string `json:"phase,omitempty"`

	// Components contains the status of each infrastructure component.
	// +optional
	Components *ComponentStatuses `json:"components,omitempty"`

	// Endpoints contains the provisioned endpoints.
	// +optional
	Endpoints *EndpointStatus `json:"endpoints,omitempty"`

	// ApiKeys contains references to the provisioned API keys.
	// +optional
	ApiKeys []ApiKeyRef `json:"apiKeys,omitempty"`

	// ObservedGeneration is the most recent generation observed.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the project's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ComponentStatuses contains the status of each infrastructure component.
type ComponentStatuses struct {
	// Postgres is the status of the PostgreSQL component.
	// +optional
	Postgres *PostgresComponentStatus `json:"postgres,omitempty"`

	// Redis is the status of the Redis component.
	// +optional
	Redis *ComponentStatus `json:"redis,omitempty"`

	// PostgREST is the status of the PostgREST component.
	// +optional
	PostgREST *ComponentStatus `json:"postgrest,omitempty"`

	// CDC is the status of the CDC Pod component.
	// +optional
	CDC *ComponentStatus `json:"cdc,omitempty"`

	// PostgresMeta is the status of the postgres-meta component.
	// +optional
	PostgresMeta *ComponentStatus `json:"postgresMeta,omitempty"`

	// GoTrue is the status of the GoTrue auth component.
	// +optional
	GoTrue *ComponentStatus `json:"gotrue,omitempty"`
}

// ComponentStatus represents the status of a single component.
type ComponentStatus struct {
	// Status is the component status (e.g., "Ready", "Provisioning", "Failed").
	Status string `json:"status,omitempty"`
}

// PostgresComponentStatus extends ComponentStatus with PostgreSQL-specific fields.
type PostgresComponentStatus struct {
	// Status is the component status.
	Status string `json:"status,omitempty"`

	// PrimaryEndpoint is the primary PostgreSQL endpoint.
	// +optional
	PrimaryEndpoint string `json:"primaryEndpoint,omitempty"`

	// PoolerEndpoint is the connection pooler endpoint.
	// +optional
	PoolerEndpoint string `json:"poolerEndpoint,omitempty"`
}

// EndpointStatus contains the provisioned endpoints.
type EndpointStatus struct {
	// RestApi is the REST API endpoint URL.
	// +optional
	RestApi string `json:"restApi,omitempty"`

	// StorageApi is the storage API endpoint URL.
	// +optional
	StorageApi string `json:"storageApi,omitempty"`

	// DbHost is the database host for direct connections.
	// +optional
	DbHost string `json:"dbHost,omitempty"`

	// DbConnectionString is the full database connection string.
	// +optional
	DbConnectionString string `json:"dbConnectionString,omitempty"`

	// AuthApi is the GoTrue auth API endpoint URL.
	// +optional
	AuthApi string `json:"authApi,omitempty"`
}

// ApiKeyRef is a reference to a provisioned API key.
type ApiKeyRef struct {
	// ID is the API key identifier.
	ID string `json:"id"`

	// Name is the API key name (e.g., "default-service-role").
	Name string `json:"name"`
}

// Project phase constants.
const (
	ProjectPhasePending      = "Pending"
	ProjectPhaseQueued       = "Queued"
	ProjectPhaseProvisioning = "Provisioning"
	ProjectPhaseReady        = "Ready"
	ProjectPhaseFailed       = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Plan",type=string,JSONPath=`.spec.plan`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Project is the Schema for the projects API.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
