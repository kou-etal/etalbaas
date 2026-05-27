package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/services/function/internal/store"
)

// =========================================================================
// A) Validation tests (nil querier)
// =========================================================================

func TestValidateFunctionName_Empty(t *testing.T) {
	if err := validateFunctionName(""); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateFunctionName_TooShort(t *testing.T) {
	if err := validateFunctionName("ab"); err == nil {
		t.Fatal("expected error for 2-char name")
	}
}

func TestValidateFunctionName_TooLong(t *testing.T) {
	long := make([]byte, maxFunctionNameLen+1)
	for i := range long {
		long[i] = 'a'
	}
	if err := validateFunctionName(string(long)); err == nil {
		t.Fatal("expected error for too-long name")
	}
}

func TestValidateFunctionName_InvalidFormat(t *testing.T) {
	invalid := []string{
		"has_underscore",
		"HasUpperCase",
		"-starts-dash",
		"ends-dash-",
		"has space",
		"has.dot",
	}
	for _, name := range invalid {
		if err := validateFunctionName(name); err == nil {
			t.Fatalf("expected error for invalid name %q", name)
		}
	}
}

func TestValidateFunctionName_Valid(t *testing.T) {
	valid := []string{
		"abc",
		"my-func",
		"my-function-123",
		"a0b",
	}
	for _, name := range valid {
		if err := validateFunctionName(name); err != nil {
			t.Fatalf("expected valid name %q, got error: %v", name, err)
		}
	}
}

func TestValidateKind_Invalid(t *testing.T) {
	if err := validateKind("invalid-kind"); err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestValidateKind_Valid(t *testing.T) {
	for _, k := range []string{"heavy-job", "heavy-deployment", "light-deployment"} {
		if err := validateKind(k); err != nil {
			t.Fatalf("expected valid kind %q, got error: %v", k, err)
		}
	}
}

func TestValidateMode_Invalid(t *testing.T) {
	if err := validateMode("invalid"); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestValidateMode_Valid(t *testing.T) {
	for _, m := range []string{"sync", "async", "stream"} {
		if err := validateMode(m); err != nil {
			t.Fatalf("expected valid mode %q, got error: %v", m, err)
		}
	}
}

func TestValidateRuntime_BothSet(t *testing.T) {
	preset := "python3.11"
	dockerfile := "FROM python:3.11"
	if err := validateRuntime(&preset, &dockerfile); err == nil {
		t.Fatal("expected error when both preset and dockerfile are set")
	}
}

func TestValidateRuntime_NeitherSet(t *testing.T) {
	if err := validateRuntime(nil, nil); err == nil {
		t.Fatal("expected error when neither preset nor dockerfile is set")
	}
}

func TestValidateRuntime_PresetOnly(t *testing.T) {
	preset := "python3.11"
	if err := validateRuntime(&preset, nil); err != nil {
		t.Fatalf("expected valid runtime, got error: %v", err)
	}
}

func TestValidateRuntime_DockerfileOnly(t *testing.T) {
	dockerfile := "FROM python:3.11"
	if err := validateRuntime(nil, &dockerfile); err != nil {
		t.Fatalf("expected valid runtime, got error: %v", err)
	}
}

func TestValidateTimeout_LightExceed(t *testing.T) {
	if err := validateTimeout(31, "light-deployment", nil); err == nil {
		t.Fatal("expected error for light-deployment timeout > 30")
	}
}

func TestValidateTimeout_HeavyCPUExceed(t *testing.T) {
	if err := validateTimeout(3601, "heavy-job", nil); err == nil {
		t.Fatal("expected error for heavy-job CPU timeout > 3600")
	}
}

func TestValidateTimeout_HeavyGPU_NoLimit(t *testing.T) {
	gpuConfig, _ := json.Marshal(GpuConfigJSON{Type: "H100", Provider: "runpod", Product: "serverless"})
	if err := validateTimeout(99999, "heavy-job", gpuConfig); err != nil {
		t.Fatalf("expected no limit for heavy+GPU, got error: %v", err)
	}
}

func TestValidateTimeout_EmptyGpuConfigNotBypass(t *testing.T) {
	// Empty GPU config {} should NOT bypass the CPU timeout limit
	if err := validateTimeout(3601, "heavy-job", []byte("{}")); err == nil {
		t.Fatal("expected error: empty GPU config should not bypass CPU timeout limit")
	}
}

func TestValidateTriggers_CronRejected(t *testing.T) {
	triggers := []TriggerJSON{{Type: "cron"}}
	data, _ := json.Marshal(triggers)
	if err := validateTriggers(data); err == nil {
		t.Fatal("expected error for cron trigger type")
	}
	if apperror.CodeOf(validateTriggers(data)) != apperror.CodeInvalidArgument {
		t.Fatal("expected CodeInvalidArgument for cron trigger")
	}
}

func TestValidateTriggers_AllowedTypes(t *testing.T) {
	triggers := []TriggerJSON{
		{Type: "database_change", DatabaseChange: &DatabaseChangeTriggerJSON{Table: "users", Events: []string{"INSERT"}}},
		{Type: "object_storage", ObjectStorage: &ObjectStorageTriggerJSON{Bucket: "uploads", Events: []string{"ObjectCreated"}}},
	}
	data, _ := json.Marshal(triggers)
	if err := validateTriggers(data); err != nil {
		t.Fatalf("expected valid triggers, got error: %v", err)
	}
}

func TestValidateTriggers_Empty(t *testing.T) {
	if err := validateTriggers(nil); err != nil {
		t.Fatalf("expected nil triggers to be valid, got error: %v", err)
	}
	if err := validateTriggers([]byte("[]")); err != nil {
		t.Fatalf("expected empty array triggers to be valid, got error: %v", err)
	}
}

func TestCreateFunction_ValidationErrors(t *testing.T) {
	svc := NewFunctionService(nil, nil)
	preset := "python3.11"

	// Empty name
	_, err := svc.CreateFunction(t.Context(), CreateFunctionParams{
		Name:          "",
		DisplayName:   "Test",
		Kind:          "light-deployment",
		Mode:          "sync",
		RuntimePreset: &preset,
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

// =========================================================================
// B) UpdateFunction merge logic tests (mock querier)
// =========================================================================

type mockQuerier struct {
	countActiveFunctionsByProjectIDFn func(ctx context.Context, projectID string) (int64, error)
	getFunctionByIDAndProjectIDFn    func(ctx context.Context, arg store.GetFunctionByIDAndProjectIDParams) (store.Function, error)
	getProjectByIDAndTenantIDFn      func(ctx context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error)
	updateFunctionFn                 func(ctx context.Context, arg store.UpdateFunctionParams) (store.Function, error)
	updateFunctionBuildStatusFn      func(ctx context.Context, arg store.UpdateFunctionBuildStatusParams) (store.Function, error)

	// Unused stubs
	createFunctionFn              func(ctx context.Context, arg store.CreateFunctionParams) (store.Function, error)
	deleteFunctionFn              func(ctx context.Context, arg store.DeleteFunctionParams) (store.Function, error)
	listFunctionsByProjectIDFn    func(ctx context.Context, arg store.ListFunctionsByProjectIDParams) ([]store.Function, error)
	listInvocationsByFunctionIDFn func(ctx context.Context, arg store.ListInvocationsByFunctionIDParams) ([]store.Invocation, error)
}

func (m *mockQuerier) CountActiveFunctionsByProjectID(ctx context.Context, projectID string) (int64, error) {
	if m.countActiveFunctionsByProjectIDFn != nil {
		return m.countActiveFunctionsByProjectIDFn(ctx, projectID)
	}
	return 0, nil
}

func (m *mockQuerier) CreateFunction(ctx context.Context, arg store.CreateFunctionParams) (store.Function, error) {
	if m.createFunctionFn != nil {
		return m.createFunctionFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetFunctionByIDAndProjectID(ctx context.Context, arg store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
	if m.getFunctionByIDAndProjectIDFn != nil {
		return m.getFunctionByIDAndProjectIDFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetProjectByIDAndTenantID(ctx context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
	if m.getProjectByIDAndTenantIDFn != nil {
		return m.getProjectByIDAndTenantIDFn(ctx, arg)
	}
	return store.Project{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListFunctionsByProjectID(ctx context.Context, arg store.ListFunctionsByProjectIDParams) ([]store.Function, error) {
	if m.listFunctionsByProjectIDFn != nil {
		return m.listFunctionsByProjectIDFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) UpdateFunction(ctx context.Context, arg store.UpdateFunctionParams) (store.Function, error) {
	if m.updateFunctionFn != nil {
		return m.updateFunctionFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func (m *mockQuerier) DeleteFunction(ctx context.Context, arg store.DeleteFunctionParams) (store.Function, error) {
	if m.deleteFunctionFn != nil {
		return m.deleteFunctionFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListInvocationsByFunctionID(ctx context.Context, arg store.ListInvocationsByFunctionIDParams) ([]store.Invocation, error) {
	if m.listInvocationsByFunctionIDFn != nil {
		return m.listInvocationsByFunctionIDFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) UpdateFunctionBuildStatus(ctx context.Context, arg store.UpdateFunctionBuildStatusParams) (store.Function, error) {
	if m.updateFunctionBuildStatusFn != nil {
		return m.updateFunctionBuildStatusFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func newCurrentFunction() store.Function {
	preset := "python3.11"
	triggersJSON, _ := json.Marshal([]TriggerJSON{{
		Type:           "database_change",
		DatabaseChange: &DatabaseChangeTriggerJSON{Table: "orders", Events: []string{"INSERT"}},
	}})
	envVarsJSON, _ := json.Marshal([]EnvVarJSON{{Name: "API_KEY", Value: "secret"}})
	return store.Function{
		ID:                  uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ProjectID:           "proj-123",
		Name:                "my-func",
		DisplayName:         "Original Name",
		Kind:                "heavy-job",
		Mode:                "sync",
		SourceType:          "git",
		SourceConfig:        mustJSON(GitSourceConfig{RepoURL: "https://github.com/test/repo", Branch: "main", Subpath: "/"}),
		RuntimePreset:       &preset,
		RuntimeRequirements: []string{"requests"},
		TimeoutSec:          60,
		Triggers:            triggersJSON,
		EnvVars:             envVarsJSON,
		Status:              "ready",
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestUpdateFunction_MergeOptionalScalar(t *testing.T) {
	current := newCurrentFunction()
	var captured store.UpdateFunctionParams

	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return store.Project{}, nil
		},
		getFunctionByIDAndProjectIDFn: func(_ context.Context, _ store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			return current, nil
		},
		updateFunctionFn: func(_ context.Context, arg store.UpdateFunctionParams) (store.Function, error) {
			captured = arg
			result := current
			result.DisplayName = arg.DisplayName
			result.TimeoutSec = arg.TimeoutSec
			return result, nil
		},
	}

	svc := NewFunctionService(q, nil)
	newName := "Updated Name"
	newTimeout := int32(120)

	_, err := svc.UpdateFunction(t.Context(), UpdateFunctionParams{
		TenantID:    uuid.New(),
		ProjectID:   "proj-123",
		ID:          current.ID,
		DisplayName: &newName,
		TimeoutSec:  &newTimeout,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify merged values
	if captured.DisplayName != "Updated Name" {
		t.Fatalf("expected DisplayName 'Updated Name', got %q", captured.DisplayName)
	}
	if captured.TimeoutSec != 120 {
		t.Fatalf("expected TimeoutSec 120, got %d", captured.TimeoutSec)
	}
	// Unchanged fields should retain current values
	if captured.Mode != "sync" {
		t.Fatalf("expected Mode 'sync' (unchanged), got %q", captured.Mode)
	}
	if captured.SourceType != "git" {
		t.Fatalf("expected SourceType 'git' (unchanged), got %q", captured.SourceType)
	}
}

func TestUpdateFunction_MergeOneofSourceSwitch(t *testing.T) {
	current := newCurrentFunction() // git source
	var captured store.UpdateFunctionParams

	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return store.Project{}, nil
		},
		getFunctionByIDAndProjectIDFn: func(_ context.Context, _ store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			return current, nil
		},
		updateFunctionFn: func(_ context.Context, arg store.UpdateFunctionParams) (store.Function, error) {
			captured = arg
			return current, nil
		},
	}

	svc := NewFunctionService(q, nil)
	inlineType := "inline"
	inlineConfig := mustJSON(InlineSourceConfig{Code: "print('hello')", Filename: "main.py"})

	_, err := svc.UpdateFunction(t.Context(), UpdateFunctionParams{
		TenantID:      uuid.New(),
		ProjectID:     "proj-123",
		ID:            current.ID,
		SourceType:    &inlineType,
		SourceConfig:  inlineConfig,
		SourceStoragePath: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.SourceType != "inline" {
		t.Fatalf("expected SourceType 'inline', got %q", captured.SourceType)
	}
	var sc InlineSourceConfig
	if err := json.Unmarshal(captured.SourceConfig, &sc); err != nil {
		t.Fatalf("failed to unmarshal source_config: %v", err)
	}
	if sc.Code != "print('hello')" {
		t.Fatalf("expected code 'print('hello')', got %q", sc.Code)
	}
}

func TestUpdateFunction_MergeOneofRuntimeSwitch(t *testing.T) {
	current := newCurrentFunction() // preset runtime
	var captured store.UpdateFunctionParams

	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return store.Project{}, nil
		},
		getFunctionByIDAndProjectIDFn: func(_ context.Context, _ store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			return current, nil
		},
		updateFunctionFn: func(_ context.Context, arg store.UpdateFunctionParams) (store.Function, error) {
			captured = arg
			return current, nil
		},
	}

	svc := NewFunctionService(q, nil)
	dockerfile := "FROM python:3.11\nRUN pip install flask"

	_, err := svc.UpdateFunction(t.Context(), UpdateFunctionParams{
		TenantID:          uuid.New(),
		ProjectID:         "proj-123",
		ID:                current.ID,
		RuntimeDockerfile: &dockerfile,
		RuntimeSet:        true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.RuntimePreset != nil {
		t.Fatalf("expected RuntimePreset nil after switch, got %q", *captured.RuntimePreset)
	}
	if captured.RuntimeDockerfile == nil || *captured.RuntimeDockerfile != dockerfile {
		t.Fatal("expected RuntimeDockerfile to be set")
	}
}

func TestUpdateFunction_MergeRepeatedReplace(t *testing.T) {
	current := newCurrentFunction() // has triggers and env_vars
	var captured store.UpdateFunctionParams

	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return store.Project{}, nil
		},
		getFunctionByIDAndProjectIDFn: func(_ context.Context, _ store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			return current, nil
		},
		updateFunctionFn: func(_ context.Context, arg store.UpdateFunctionParams) (store.Function, error) {
			captured = arg
			return current, nil
		},
	}

	svc := NewFunctionService(q, nil)
	newTriggers := mustJSON([]TriggerJSON{{
		Type:          "object_storage",
		ObjectStorage: &ObjectStorageTriggerJSON{Bucket: "images", Events: []string{"ObjectCreated"}},
	}})
	newEnvVars := mustJSON([]EnvVarJSON{{Name: "NEW_VAR", Value: "new-value"}})

	_, err := svc.UpdateFunction(t.Context(), UpdateFunctionParams{
		TenantID:    uuid.New(),
		ProjectID:   "proj-123",
		ID:          current.ID,
		Triggers:    newTriggers,
		TriggersSet: true,
		EnvVars:     newEnvVars,
		EnvVarsSet:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Triggers should be fully replaced
	var triggers []TriggerJSON
	if err := json.Unmarshal(captured.Triggers, &triggers); err != nil {
		t.Fatalf("failed to unmarshal triggers: %v", err)
	}
	if len(triggers) != 1 || triggers[0].Type != "object_storage" {
		t.Fatalf("expected 1 object_storage trigger, got %+v", triggers)
	}

	// EnvVars should be fully replaced
	var envVars []EnvVarJSON
	if err := json.Unmarshal(captured.EnvVars, &envVars); err != nil {
		t.Fatalf("failed to unmarshal env_vars: %v", err)
	}
	if len(envVars) != 1 || envVars[0].Name != "NEW_VAR" {
		t.Fatalf("expected 1 env_var 'NEW_VAR', got %+v", envVars)
	}
}

func TestUpdateFunction_MergeExclusiveConstraintDetected(t *testing.T) {
	// Current has preset runtime. Try to set dockerfile without clearing preset → error.
	current := newCurrentFunction()
	preset := "python3.11"
	dockerfile := "FROM python:3.11"

	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return store.Project{}, nil
		},
		getFunctionByIDAndProjectIDFn: func(_ context.Context, _ store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			return current, nil
		},
	}

	svc := NewFunctionService(q, nil)

	// Send both preset and dockerfile via RuntimeSet
	_, err := svc.UpdateFunction(t.Context(), UpdateFunctionParams{
		TenantID:          uuid.New(),
		ProjectID:         "proj-123",
		ID:                current.ID,
		RuntimePreset:     &preset,
		RuntimeDockerfile: &dockerfile,
		RuntimeSet:        true,
	})
	if err == nil {
		t.Fatal("expected error for exclusive runtime constraint")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}
