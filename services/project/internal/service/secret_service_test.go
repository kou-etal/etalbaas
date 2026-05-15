package service

import (
	"testing"

	"github.com/kou-etal/etalbaas/pkg/apperror"
)

func TestCreateSecret_EmptyName(t *testing.T) {
	svc := NewSecretService(nil, nil)
	_, err := svc.CreateSecret(t.Context(), CreateSecretParams{
		Name:  "",
		Value: "some-value",
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateSecret_NameTooLong(t *testing.T) {
	svc := NewSecretService(nil, nil)
	longName := make([]byte, maxSecretNameLen+1)
	for i := range longName {
		longName[i] = 'A'
	}
	_, err := svc.CreateSecret(t.Context(), CreateSecretParams{
		Name:  string(longName),
		Value: "some-value",
	})
	if err == nil {
		t.Fatal("expected error for long name")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateSecret_InvalidNameFormat(t *testing.T) {
	tests := []string{
		"lowercase",
		"has-dash",
		"has space",
		"has.dot",
		"mixedCase",
	}
	svc := NewSecretService(nil, nil)
	for _, name := range tests {
		_, err := svc.CreateSecret(t.Context(), CreateSecretParams{
			Name:  name,
			Value: "some-value",
		})
		if err == nil {
			t.Fatalf("expected error for invalid name %q", name)
		}
		if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for %q, got %v", name, apperror.CodeOf(err))
		}
	}
}

func TestCreateSecret_ValidNames(t *testing.T) {
	tests := []string{
		"OPENAI_API_KEY",
		"STRIPE_WEBHOOK_SECRET",
		"MY_VAR_123",
		"A",
	}
	for _, name := range tests {
		if err := validateSecretName(name); err != nil {
			t.Fatalf("expected valid name %q, got error: %v", name, err)
		}
	}
}

func TestCreateSecret_EmptyValue(t *testing.T) {
	svc := NewSecretService(nil, nil)
	_, err := svc.CreateSecret(t.Context(), CreateSecretParams{
		Name:  "VALID_NAME",
		Value: "",
	})
	if err == nil {
		t.Fatal("expected error for empty value")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateSecret_ValueTooLong(t *testing.T) {
	svc := NewSecretService(nil, nil)
	longValue := make([]byte, maxSecretValueLen+1)
	for i := range longValue {
		longValue[i] = 'x'
	}
	_, err := svc.CreateSecret(t.Context(), CreateSecretParams{
		Name:  "VALID_NAME",
		Value: string(longValue),
	})
	if err == nil {
		t.Fatal("expected error for long value")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestCreateSecret_DescriptionTooLong(t *testing.T) {
	svc := NewSecretService(nil, nil)
	longDesc := make([]byte, maxSecretDescriptionLen+1)
	for i := range longDesc {
		longDesc[i] = 'a'
	}
	_, err := svc.CreateSecret(t.Context(), CreateSecretParams{
		Name:        "VALID_NAME",
		Value:       "some-value",
		Description: string(longDesc),
	})
	if err == nil {
		t.Fatal("expected error for long description")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}

func TestUpdateSecretValue_EmptyValue(t *testing.T) {
	svc := NewSecretService(nil, nil)
	_, err := svc.UpdateSecretValue(t.Context(), [16]byte{}, "proj-123", [16]byte{}, "")
	if err == nil {
		t.Fatal("expected error for empty value")
	}
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", apperror.CodeOf(err))
	}
}
