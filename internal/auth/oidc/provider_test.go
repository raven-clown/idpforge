package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testKeySet(t *testing.T) *KeySet {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return &KeySet{KeyID: "test-key", PrivateKey: priv}
}

func TestVerifyPKCE(t *testing.T) {
	// RFC 7636 Appendix B test vector.
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if !verifyPKCE(challenge, verifier) {
		t.Error("expected matching verifier/challenge pair to verify")
	}
	if verifyPKCE(challenge, "wrong-verifier") {
		t.Error("expected mismatched verifier to fail")
	}
	if verifyPKCE("", verifier) {
		t.Error("expected empty challenge to fail")
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	if hashToken("token-a") != hashToken("token-a") {
		t.Error("hashToken should be deterministic")
	}
	if hashToken("token-a") == hashToken("token-b") {
		t.Error("hashToken should differ for different inputs")
	}
}

func TestSignAndParseAccessToken(t *testing.T) {
	keys := testKeySet(t)
	p := &Provider{keys: keys}

	token, err := p.sign(jwt.MapClaims{
		"sub": "user-123",
		"aud": "client-abc",
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	claims, err := p.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims["sub"] != "user-123" {
		t.Errorf("sub = %v, want user-123", claims["sub"])
	}
}

func TestParseAccessTokenRejectsForgedKey(t *testing.T) {
	real := testKeySet(t)
	forged := testKeySet(t)

	p := &Provider{keys: forged}
	token, err := p.sign(jwt.MapClaims{"sub": "attacker", "exp": jwt.NewNumericDate(time.Now().Add(time.Hour))})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	realProvider := &Provider{keys: real}
	if _, err := realProvider.ParseAccessToken(token); err == nil {
		t.Error("expected a token signed by a different key to fail verification")
	}
}

func TestJWKSShape(t *testing.T) {
	p := &Provider{keys: testKeySet(t)}
	jwks := p.JWKS()
	keys, ok := jwks["keys"].([]interface{})
	if !ok || len(keys) != 1 {
		t.Fatalf("expected JWKS to contain exactly one key, got %v", jwks)
	}
}
