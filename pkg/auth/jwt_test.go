package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/kou-etal/etalbaas/pkg/auth"
)

var testSigningKey = []byte("test-secret-key")

func createTestToken(t *testing.T, userID string, expiry time.Time) string {
	t.Helper()
	claims := &auth.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(testSigningKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenStr
}

func TestVerifyToken_Valid(t *testing.T) {
	tokenStr := createTestToken(t, "user-123", time.Now().Add(time.Hour))

	claims, err := auth.VerifyToken(tokenStr, testSigningKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("got user_id %q, want %q", claims.UserID, "user-123")
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	tokenStr := createTestToken(t, "user-123", time.Now().Add(-time.Hour))

	_, err := auth.VerifyToken(tokenStr, testSigningKey)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyToken_WrongKey(t *testing.T) {
	tokenStr := createTestToken(t, "user-123", time.Now().Add(time.Hour))

	_, err := auth.VerifyToken(tokenStr, []byte("wrong-key"))
	if err == nil {
		t.Fatal("expected error for wrong signing key")
	}
}

func TestVerifyToken_EmptyUserID(t *testing.T) {
	tokenStr := createTestToken(t, "", time.Now().Add(time.Hour))

	_, err := auth.VerifyToken(tokenStr, testSigningKey)
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestVerifyToken_InvalidToken(t *testing.T) {
	_, err := auth.VerifyToken("not-a-valid-token", testSigningKey)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}
