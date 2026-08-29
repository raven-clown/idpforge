package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/testutil"
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

func TestHashClientSecretDeterministic(t *testing.T) {
	if hashClientSecret("secret-a") != hashClientSecret("secret-a") {
		t.Error("hashClientSecret should be deterministic")
	}
	if hashClientSecret("secret-a") == hashClientSecret("secret-b") {
		t.Error("hashClientSecret should differ for different inputs")
	}
	// Must match `printf '%s' "$SECRET" | openssl dgst -sha256 | awk
	// '{print $2}'`, since that's how scripts/add-app.sh computes the
	// stored client_secret_hash -- a known SHA-256 test vector pins the
	// exact hex encoding (lowercase, no separators, no prefix).
	if got := hashClientSecret("hello"); got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("hashClientSecret(%q) = %q, want the standard sha256 hex digest", "hello", got)
	}
}

func seedApplication(t *testing.T, ctx context.Context, store *ClientStore, clientID, secretHash string) {
	t.Helper()
	config := fmt.Sprintf(`{"client_id":%q,"client_secret_hash":%q,"redirect_uris":["https://app.example.com/callback"],"allowed_scopes":["openid"]}`, clientID, secretHash)
	if _, err := store.db.ExecContext(ctx, `INSERT INTO applications (id, name, protocol, config, enabled) VALUES (?, ?, 'oidc', ?, ?)`,
		clientID, "test app", config, true); err != nil {
		t.Fatalf("seed application: %v", err)
	}
}

func TestAuthenticateClientConfidentialRejectsWrongSecret(t *testing.T) {
	database := testutil.OpenTestDB(t)
	store := NewClientStore(database)
	ctx := context.Background()
	seedApplication(t, ctx, store, "confidential-app", hashClientSecret("correct-secret"))

	p := &Provider{clients: store}

	if err := p.authenticateClient(ctx, "confidential-app", "wrong-secret"); err == nil {
		t.Error("expected wrong client_secret to be rejected")
	}
	if err := p.authenticateClient(ctx, "confidential-app", "correct-secret"); err != nil {
		t.Errorf("expected correct client_secret to be accepted, got: %v", err)
	}
}

func TestAuthenticateClientPublicNeedsNoSecret(t *testing.T) {
	database := testutil.OpenTestDB(t)
	store := NewClientStore(database)
	ctx := context.Background()
	seedApplication(t, ctx, store, "public-spa", "") // no secret hash: public client

	p := &Provider{clients: store}

	if err := p.authenticateClient(ctx, "public-spa", ""); err != nil {
		t.Errorf("expected a public client (no secret hash) to authenticate with no secret, got: %v", err)
	}
}

func TestClientCredentialsGrantsExactlyAllowedScopes(t *testing.T) {
	database := testutil.OpenTestDB(t)
	store := NewClientStore(database)
	ctx := context.Background()

	appConfig := fmt.Sprintf(`{"client_id":%q,"client_secret_hash":%q,"redirect_uris":[],"allowed_scopes":["reports:read","reports:export"]}`,
		"m2m-client", hashClientSecret("m2m-secret"))
	if _, err := store.db.ExecContext(ctx, `INSERT INTO applications (id, name, protocol, config, enabled) VALUES (?, ?, 'oidc', ?, ?)`,
		"m2m-client", "m2m app", appConfig, true); err != nil {
		t.Fatalf("seed application: %v", err)
	}

	p := &Provider{clients: store, keys: testKeySet(t), cfg: config.OIDCConfig{
		Issuer:         "https://idp.example.com",
		AccessTokenTTL: time.Hour,
		IDTokenTTL:     time.Hour,
	}}

	if _, err := p.ClientCredentials(ctx, "m2m-client", "wrong-secret"); err == nil {
		t.Error("expected wrong client_secret to be rejected")
	}

	resp, err := p.ClientCredentials(ctx, "m2m-client", "m2m-secret")
	if err != nil {
		t.Fatalf("ClientCredentials: %v", err)
	}
	if resp.RefreshToken != "" {
		t.Error("client_credentials should never issue a refresh token")
	}
	if resp.Scope != "reports:read reports:export" {
		t.Errorf("scope = %q, want the client's allowed_scopes verbatim", resp.Scope)
	}

	claims, err := p.ParseAccessToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims["sub"] != "m2m-client" {
		t.Errorf("sub = %v, want the client_id itself", claims["sub"])
	}
}

func TestAuthenticateClientUnknownClientRejected(t *testing.T) {
	database := testutil.OpenTestDB(t)
	store := NewClientStore(database)
	p := &Provider{clients: store}

	if err := p.authenticateClient(context.Background(), "does-not-exist", "anything"); err == nil {
		t.Error("expected an unregistered client_id to be rejected")
	}
}
