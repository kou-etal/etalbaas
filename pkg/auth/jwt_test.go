package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/kou-etal/etalbaas/pkg/auth"
)

var testRSAKey *rsa.PrivateKey

func init() {
	var err error
	testRSAKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("generate test RSA key: " + err.Error())
	}
}

func createTestToken(t *testing.T, userID string, expiry time.Time) string {
	t.Helper()
	claims := &auth.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(testRSAKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenStr
}

func TestVerifyToken_Valid(t *testing.T) {
	tokenStr := createTestToken(t, "user-123", time.Now().Add(time.Hour))

	claims, err := auth.VerifyToken(tokenStr, &testRSAKey.PublicKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("got user_id %q, want %q", claims.UserID, "user-123")
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	tokenStr := createTestToken(t, "user-123", time.Now().Add(-time.Hour))

	_, err := auth.VerifyToken(tokenStr, &testRSAKey.PublicKey)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyToken_WrongKey(t *testing.T) {
	tokenStr := createTestToken(t, "user-123", time.Now().Add(time.Hour))

	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	_, err := auth.VerifyToken(tokenStr, &otherKey.PublicKey)
	if err == nil {
		t.Fatal("expected error for wrong signing key")
	}
}

func TestVerifyToken_EmptyUserID(t *testing.T) {
	tokenStr := createTestToken(t, "", time.Now().Add(time.Hour))

	_, err := auth.VerifyToken(tokenStr, &testRSAKey.PublicKey)
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestVerifyToken_InvalidToken(t *testing.T) {
	_, err := auth.VerifyToken("not-a-valid-token", &testRSAKey.PublicKey)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestVerifyToken_RejectHS256(t *testing.T) {
	// HS256 tokens must be rejected (Key Confusion Attack prevention).
	hmacKey := []byte("test-secret-key")
	claims := &auth.Claims{
		UserID: "user-hs256",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(hmacKey)
	if err != nil {
		t.Fatalf("sign HS256 token: %v", err)
	}

	_, err = auth.VerifyToken(tokenStr, &testRSAKey.PublicKey)
	if err == nil {
		t.Fatal("expected error: HS256 token should be rejected")
	}
}
