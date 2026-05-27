package resources

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const (
	jwtKID = "key1"
	jwtAlg = "RS256"
)

// GenerateRSAKeyPair generates a 2048-bit RSA key pair.
func GenerateRSAKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// rsaJWK represents an RSA JSON Web Key.
type rsaJWK struct {
	Kty    string   `json:"kty"`
	Kid    string   `json:"kid"`
	Use    string   `json:"use"`
	Alg    string   `json:"alg"`
	KeyOps []string `json:"key_ops,omitempty"`
	N      string   `json:"n"`
	E      string   `json:"e"`
	D      string   `json:"d,omitempty"`
	P      string   `json:"p,omitempty"`
	Q      string   `json:"q,omitempty"`
	Dp     string   `json:"dp,omitempty"`
	Dq     string   `json:"dq,omitempty"`
	Qi     string   `json:"qi,omitempty"`
}

// rsaJWKSet represents a JWK Set containing RSA keys.
type rsaJWKSet struct {
	Keys []rsaJWK `json:"keys"`
}

// base64urlEncodeBigInt encodes a big.Int as base64url without padding.
func base64urlEncodeBigInt(n *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(n.Bytes())
}

// base64urlEncodeInt encodes an integer as base64url without padding.
func base64urlEncodeInt(n int) string {
	b := big.NewInt(int64(n))
	return base64urlEncodeBigInt(b)
}

// PrivateKeyToJWKSet converts an RSA private key to a JSON array of JWK objects.
// GoTrue's GOTRUE_JWT_KEYS expects a JSON array ([{...}]), not a JWK Set ({"keys":[...]}).
// Used for GoTrue's GOTRUE_JWT_KEYS environment variable.
func PrivateKeyToJWKSet(key *rsa.PrivateKey) ([]byte, error) {
	jwk := rsaJWK{
		Kty:    "RSA",
		Kid:    jwtKID,
		Use:    "sig",
		Alg:    jwtAlg,
		KeyOps: []string{"sign"},
		N:      base64urlEncodeBigInt(key.N),
		E:      base64urlEncodeInt(key.E),
		D:      base64urlEncodeBigInt(key.D),
		P:      base64urlEncodeBigInt(key.Primes[0]),
		Q:      base64urlEncodeBigInt(key.Primes[1]),
		Dp:     base64urlEncodeBigInt(key.Precomputed.Dp),
		Dq:     base64urlEncodeBigInt(key.Precomputed.Dq),
		Qi:     base64urlEncodeBigInt(key.Precomputed.Qinv),
	}

	// GoTrue expects a JSON array of JWK objects, not a JWK Set wrapper.
	return json.Marshal([]rsaJWK{jwk})
}

// PublicKeyToJWK converts an RSA public key to a single JWK JSON (public only).
// Used for PostgREST's PGRST_JWT_SECRET environment variable.
func PublicKeyToJWK(key *rsa.PublicKey) ([]byte, error) {
	jwk := rsaJWK{
		Kty: "RSA",
		Kid: jwtKID,
		Use: "sig",
		Alg: jwtAlg,
		N:   base64urlEncodeBigInt(key.N),
		E:   base64urlEncodeInt(key.E),
	}
	return json.Marshal(jwk)
}

// jwtHeader is the JWT header for RS256.
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

// SignRS256JWT signs a JWT with RS256 using the given RSA private key.
// Used to pre-generate service_role and anon JWTs stored in Kubernetes Secrets.
func SignRS256JWT(claims map[string]interface{}, key *rsa.PrivateKey, kid string) (string, error) {
	// Header
	header := jwtHeader{
		Alg: jwtAlg,
		Typ: "JWT",
		Kid: kid,
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal JWT header: %w", err)
	}

	// Claims
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal JWT claims: %w", err)
	}

	// Encode
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerB64 + "." + claimsB64

	// Sign with RS256 (RSASSA-PKCS1-v1_5 with SHA-256)
	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + signatureB64, nil
}

// BuildServiceRoleClaims builds JWT claims for service_role.
func BuildServiceRoleClaims() map[string]interface{} {
	now := time.Now()
	return map[string]interface{}{
		"role": "service_role",
		"iss":  "etalbaas",
		"aud":  "authenticated",
		"iat":  now.Unix(),
		"exp":  now.Add(365 * 24 * time.Hour).Unix(), // 1 year — rotate by deleting the Secret
	}
}

// BuildAnonClaims builds JWT claims for anon.
func BuildAnonClaims() map[string]interface{} {
	now := time.Now()
	return map[string]interface{}{
		"role": "anon",
		"iss":  "etalbaas",
		"aud":  "authenticated",
		"iat":  now.Unix(),
		"exp":  now.Add(365 * 24 * time.Hour).Unix(), // 1 year — rotate by deleting the Secret
	}
}

// ParseRSAPrivateKeyFromJWKSet extracts an RSA private key from a JWK JSON.
// Accepts both JSON array format ([{...}]) and JWK Set format ({"keys":[...]}).
// Used to recover the key from an existing gotrue-jwt-keys Secret.
func ParseRSAPrivateKeyFromJWKSet(data []byte) (*rsa.PrivateKey, error) {
	var keys []rsaJWK

	// Try JSON array first (GoTrue format)
	if err := json.Unmarshal(data, &keys); err != nil {
		// Fall back to JWK Set format
		var set rsaJWKSet
		if err2 := json.Unmarshal(data, &set); err2 != nil {
			return nil, fmt.Errorf("unmarshal JWK data (tried array and set): array=%w, set=%w", err, err2)
		}
		keys = set.Keys
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("JWK data is empty")
	}

	jwk := keys[0]
	if jwk.Kty != "RSA" {
		return nil, fmt.Errorf("unexpected key type: %s", jwk.Kty)
	}

	n, err := base64urlDecodeBigInt(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	e, err := base64urlDecodeInt(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	d, err := base64urlDecodeBigInt(jwk.D)
	if err != nil {
		return nil, fmt.Errorf("decode d: %w", err)
	}
	p, err := base64urlDecodeBigInt(jwk.P)
	if err != nil {
		return nil, fmt.Errorf("decode p: %w", err)
	}
	q, err := base64urlDecodeBigInt(jwk.Q)
	if err != nil {
		return nil, fmt.Errorf("decode q: %w", err)
	}

	key := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: n,
			E: e,
		},
		D:      d,
		Primes: []*big.Int{p, q},
	}
	key.Precompute()

	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("invalid RSA key: %w", err)
	}

	return key, nil
}

func base64urlDecodeBigInt(s string) (*big.Int, error) {
	// Normalize: add padding if needed
	s = strings.TrimRight(s, "=")
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(b), nil
}

func base64urlDecodeInt(s string) (int, error) {
	n, err := base64urlDecodeBigInt(s)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
