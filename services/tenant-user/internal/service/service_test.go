package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/service"
)

func TestUpdateProfile_NoFields(t *testing.T) {
	svc := service.New(nil)
	_, err := svc.UpdateProfile(context.Background(), uuid.New(), nil, nil)
	if err == nil {
		t.Fatal("expected error for no fields")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestUpdateProfile_EmptyDisplayName(t *testing.T) {
	svc := service.New(nil)
	empty := ""
	_, err := svc.UpdateProfile(context.Background(), uuid.New(), &empty, nil)
	if err == nil {
		t.Fatal("expected error for empty display_name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestUpdateProfile_DisplayNameTooLong(t *testing.T) {
	svc := service.New(nil)
	long := strings.Repeat("a", 101)
	_, err := svc.UpdateProfile(context.Background(), uuid.New(), &long, nil)
	if err == nil {
		t.Fatal("expected error for long display_name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestUpdateProfile_AvatarURLTooLong(t *testing.T) {
	svc := service.New(nil)
	long := strings.Repeat("h", 2049)
	_, err := svc.UpdateProfile(context.Background(), uuid.New(), nil, &long)
	if err == nil {
		t.Fatal("expected error for long avatar_url")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}
