package service

import (
	"testing"
	"time"

	"github.com/kou-etal/etalbaas/pkg/apperror"
)

func TestCreateProject_EmptyDisplayName(t *testing.T) {
	svc := NewProjectService(nil)
	_, err := svc.CreateProject(t.Context(), CreateProjectParams{
		DisplayName: "",
	})
	if err == nil {
		t.Fatal("expected error for empty display_name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateProject_DisplayNameTooLong(t *testing.T) {
	svc := NewProjectService(nil)
	longName := make([]byte, maxDisplayNameLen+1)
	for i := range longName {
		longName[i] = 'a'
	}
	_, err := svc.CreateProject(t.Context(), CreateProjectParams{
		DisplayName: string(longName),
	})
	if err == nil {
		t.Fatal("expected error for long display_name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateProject_DescriptionTooLong(t *testing.T) {
	svc := NewProjectService(nil)
	longDesc := make([]byte, maxDescriptionLen+1)
	for i := range longDesc {
		longDesc[i] = 'a'
	}
	_, err := svc.CreateProject(t.Context(), CreateProjectParams{
		DisplayName: "valid name",
		Description: string(longDesc),
	})
	if err == nil {
		t.Fatal("expected error for long description")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateProject_InvalidExtension(t *testing.T) {
	svc := NewProjectService(nil)
	_, err := svc.CreateProject(t.Context(), CreateProjectParams{
		DisplayName:        "test",
		PostgresExtensions: []string{"invalid_ext"},
	})
	if err == nil {
		t.Fatal("expected error for invalid extension")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateProject_PostgrestWithoutPostgres(t *testing.T) {
	svc := NewProjectService(nil)
	_, err := svc.CreateProject(t.Context(), CreateProjectParams{
		DisplayName:      "test",
		PostgrestEnabled: true,
		PostgresEnabled:  false,
	})
	if err == nil {
		t.Fatal("expected error for postgrest without postgres")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateApiKey_EmptyName(t *testing.T) {
	svc := NewProjectService(nil)
	_, err := svc.CreateApiKey(t.Context(), CreateApiKeyParams{
		Name: "",
		Role: "anon",
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateApiKey_NameTooLong(t *testing.T) {
	svc := NewProjectService(nil)
	longName := make([]byte, maxApiKeyNameLen+1)
	for i := range longName {
		longName[i] = 'a'
	}
	_, err := svc.CreateApiKey(t.Context(), CreateApiKeyParams{
		Name: string(longName),
		Role: "anon",
	})
	if err == nil {
		t.Fatal("expected error for long name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateApiKey_InvalidRole(t *testing.T) {
	svc := NewProjectService(nil)
	_, err := svc.CreateApiKey(t.Context(), CreateApiKeyParams{
		Name: "my-key",
		Role: "admin",
	})
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateApiKey_NegativeExpiresInDays(t *testing.T) {
	svc := NewProjectService(nil)
	neg := int32(-1)
	_, err := svc.CreateApiKey(t.Context(), CreateApiKeyParams{
		Name:          "my-key",
		Role:          "anon",
		ExpiresInDays: &neg,
	})
	if err == nil {
		t.Fatal("expected error for negative expires_in_days")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateProject_ExtensionsWithoutPostgres(t *testing.T) {
	svc := NewProjectService(nil)
	_, err := svc.CreateProject(t.Context(), CreateProjectParams{
		DisplayName:        "test",
		PostgresEnabled:    false,
		PostgresExtensions: []string{"pgvector"},
	})
	if err == nil {
		t.Fatal("expected error for extensions without postgres")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestGenerateProjectID(t *testing.T) {
	id, err := generateProjectID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(id) != projectIDLen {
		t.Fatalf("expected length %d, got %d", projectIDLen, len(id))
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			t.Fatalf("unexpected character in project ID: %c", c)
		}
	}
}

func TestComputeExpiresAt_Nil(t *testing.T) {
	result := computeExpiresAt(nil)
	if !result.Valid {
		t.Fatal("expected valid timestamp for nil input")
	}
	// Should be approximately 90 days from now
	expected := 90
	days := int(result.Time.Sub(time.Now()).Hours() / 24)
	if days < expected-1 || days > expected+1 {
		t.Fatalf("expected ~%d days, got %d", expected, days)
	}
}

func TestComputeExpiresAt_Zero(t *testing.T) {
	zero := int32(0)
	result := computeExpiresAt(&zero)
	if !result.Valid {
		t.Fatal("expected valid timestamp for zero input")
	}
	if result.Time.Year() != noExpiryYear {
		t.Fatalf("expected year %d, got %d", noExpiryYear, result.Time.Year())
	}
}

func TestComputeExpiresAt_Positive(t *testing.T) {
	days := int32(30)
	result := computeExpiresAt(&days)
	if !result.Valid {
		t.Fatal("expected valid timestamp for positive input")
	}
	expected := 30
	actualDays := int(result.Time.Sub(time.Now()).Hours() / 24)
	if actualDays < expected-1 || actualDays > expected+1 {
		t.Fatalf("expected ~%d days, got %d", expected, actualDays)
	}
}
