package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kou-etal/etalbaas/services/storage/internal/metastore"
)

// mockMetaQuerier implements metastore.Querier for testing.
type mockMetaQuerier struct {
	getProjectFn func(ctx context.Context, arg metastore.GetProjectByIDAndTenantIDParams) (metastore.GetProjectByIDAndTenantIDRow, error)
}

func (m *mockMetaQuerier) GetProjectByIDAndTenantID(ctx context.Context, arg metastore.GetProjectByIDAndTenantIDParams) (metastore.GetProjectByIDAndTenantIDRow, error) {
	if m.getProjectFn != nil {
		return m.getProjectFn(ctx, arg)
	}
	return metastore.GetProjectByIDAndTenantIDRow{}, pgx.ErrNoRows
}

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "my-bucket", false},
		{"valid min length", "abc", false},
		{"valid all numbers", "123", false},
		{"too short", "ab", true},
		{"starts with hyphen", "-bucket", true},
		{"ends with hyphen", "bucket-", true},
		{"uppercase", "MyBucket", true},
		{"spaces", "my bucket", true},
		{"consecutive hyphens", "my--bucket", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBucketName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBucketName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAccessLevel(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"public", false},
		{"protected", false},
		{"private", false},
		{"", true},
		{"Public", true},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validateAccessLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAccessLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMIMETypes(t *testing.T) {
	tests := []struct {
		name    string
		types   []string
		wantErr bool
	}{
		{"valid", []string{"image/png", "image/jpeg"}, false},
		{"empty list", nil, false},
		{"empty string", []string{""}, true},
		{"no slash", []string{"png"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMIMETypes(tt.types)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMIMETypes(%v) error = %v, wantErr %v", tt.types, err, tt.wantErr)
			}
		})
	}
}

func TestBucketService_CreateBucket_ValidationErrors(t *testing.T) {
	meta := &mockMetaQuerier{
		getProjectFn: func(_ context.Context, _ metastore.GetProjectByIDAndTenantIDParams) (metastore.GetProjectByIDAndTenantIDRow, error) {
			return metastore.GetProjectByIDAndTenantIDRow{ID: "proj1", TenantID: uuid.New()}, nil
		},
	}

	// poolMgr is nil — we expect validation to fail before it's used.
	svc := NewBucketService(meta, nil)

	tests := []struct {
		name   string
		params CreateBucketParams
	}{
		{
			"invalid bucket name",
			CreateBucketParams{
				TenantID:    uuid.New(),
				ProjectID:   "proj1",
				Name:        "AB", // invalid
				AccessLevel: "public",
			},
		},
		{
			"invalid access level",
			CreateBucketParams{
				TenantID:    uuid.New(),
				ProjectID:   "proj1",
				Name:        "valid-bucket",
				AccessLevel: "invalid",
			},
		},
		{
			"invalid MIME types",
			CreateBucketParams{
				TenantID:         uuid.New(),
				ProjectID:        "proj1",
				Name:             "valid-bucket",
				AccessLevel:      "public",
				AllowedMimeTypes: []string{"noslash"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateBucket(context.Background(), tt.params)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestBucketService_OwnershipDenied(t *testing.T) {
	meta := &mockMetaQuerier{} // default returns ErrNoRows
	svc := NewBucketService(meta, nil)

	_, err := svc.CreateBucket(context.Background(), CreateBucketParams{
		TenantID:    uuid.New(),
		ProjectID:   "nonexistent",
		Name:        "valid-bucket",
		AccessLevel: "public",
	})
	if err == nil {
		t.Fatal("expected error for ownership check, got nil")
	}
}

func TestServiceRoleClaims(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	claims := serviceRoleClaims(id)
	want := `{"role":"service_role","sub":"00000000-0000-0000-0000-000000000001"}`
	if string(claims) != want {
		t.Errorf("serviceRoleClaims = %s, want %s", claims, want)
	}
}
