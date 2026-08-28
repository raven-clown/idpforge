// Package oidc implements IdpForge as an OpenID Connect provider: apps like
// GitLab, Harbor, Rancher, Grafana, Jenkins, Vault, NiFi, and MinIO point
// their native OIDC config at this service instead of the other way around.
package oidc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/raven-clown/idpforge/internal/cache"
	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/rbac"
)

type Provider struct {
	cfg     config.OIDCConfig
	keys    *KeySet
	clients *ClientStore
	cache   cache.Cache
	rbac    *rbac.Resolver
}

func NewProvider(cfg config.OIDCConfig, keys *KeySet, clients *ClientStore, c cache.Cache, resolver *rbac.Resolver) *Provider {
	return &Provider{cfg: cfg, keys: keys, clients: clients, cache: c, rbac: resolver}
}

type AuthorizeRequest struct {
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              string
}

// Authorize validates the request against the registered client and issues
// a short-lived authorization code bound to the PKCE challenge, per
// RFC 7636. Plain "response_type=code" without PKCE is rejected.
func (p *Provider) Authorize(ctx context.Context, req AuthorizeRequest) (code string, err error) {
	client, err := p.clients.Get(ctx, req.ClientID)
	if err != nil {
		return "", fmt.Errorf("unknown client")
	}
	if !client.AllowsRedirect(req.RedirectURI) {
		return "", fmt.Errorf("redirect_uri not registered for client")
	}
	if req.CodeChallenge == "" || req.CodeChallengeMethod != "S256" {
		return "", fmt.Errorf("PKCE (S256) is required")
	}

	code = uuid.NewString()
	record := authCode{
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		UserID:              req.UserID,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	if err := p.cache.Set(ctx, "oidc:code:"+code, string(encoded), 2*time.Minute); err != nil {
		return "", err
	}
	return code, nil
}

type authCode struct {
	ClientID            string
	RedirectURI         string
	Scope               string
	UserID              string
	CodeChallenge       string
	CodeChallengeMethod string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// ExchangeCode implements the authorization_code grant with mandatory PKCE
// verifier check.
func (p *Provider) ExchangeCode(ctx context.Context, clientID, code, redirectURI, codeVerifier string) (*TokenResponse, error) {
	raw, ok, err := p.cache.Get(ctx, "oidc:code:"+code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid or expired code")
	}
	_ = p.cache.Delete(ctx, "oidc:code:"+code) // single use

	var record authCode
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, err
	}
	if record.ClientID != clientID || record.RedirectURI != redirectURI {
		return nil, fmt.Errorf("code does not match client/redirect_uri")
	}
	if !verifyPKCE(record.CodeChallenge, codeVerifier) {
		return nil, fmt.Errorf("invalid code_verifier")
	}

	return p.issueTokens(ctx, clientID, record.UserID, record.Scope, true)
}

// RefreshTokens implements the refresh_token grant.
func (p *Provider) RefreshTokens(ctx context.Context, clientID, refreshToken string) (*TokenResponse, error) {
	key := "oidc:refresh:" + hashToken(refreshToken)
	raw, ok, err := p.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}
	var record struct {
		ClientID string
		UserID   string
		Scope    string
	}
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, err
	}
	if record.ClientID != clientID {
		return nil, fmt.Errorf("refresh token does not match client")
	}
	_ = p.cache.Delete(ctx, key) // rotate on use

	return p.issueTokens(ctx, clientID, record.UserID, record.Scope, true)
}

func (p *Provider) issueTokens(ctx context.Context, clientID, userID, scope string, withRefresh bool) (*TokenResponse, error) {
	now := time.Now().UTC()

	var roleNames []string
	if p.rbac != nil {
		if resolved, err := p.rbac.Resolve(ctx, userID); err == nil {
			roleNames = resolved.RoleNames
		}
	}

	accessClaims := jwt.MapClaims{
		"iss":   p.cfg.Issuer,
		"sub":   userID,
		"aud":   clientID,
		"scope": scope,
		"roles": roleNames,
		"iat":   now.Unix(),
		"exp":   now.Add(p.cfg.AccessTokenTTL).Unix(),
	}
	accessToken, err := p.sign(accessClaims)
	if err != nil {
		return nil, err
	}

	idClaims := jwt.MapClaims{
		"iss": p.cfg.Issuer,
		"sub": userID,
		"aud": clientID,
		"iat": now.Unix(),
		"exp": now.Add(p.cfg.IDTokenTTL).Unix(),
	}
	idToken, err := p.sign(idClaims)
	if err != nil {
		return nil, err
	}

	resp := &TokenResponse{
		AccessToken: accessToken,
		IDToken:     idToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(p.cfg.AccessTokenTTL.Seconds()),
		Scope:       scope,
	}

	if withRefresh {
		refreshToken := uuid.NewString()
		record, _ := json.Marshal(struct {
			ClientID string
			UserID   string
			Scope    string
		}{clientID, userID, scope})
		key := "oidc:refresh:" + hashToken(refreshToken)
		if err := p.cache.Set(ctx, key, string(record), p.cfg.RefreshTokenTTL); err != nil {
			return nil, err
		}
		resp.RefreshToken = refreshToken
	}

	return resp, nil
}

func (p *Provider) sign(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(p.keys.SigningMethod(), claims)
	token.Header["kid"] = p.keys.KeyID
	return token.SignedString(p.keys.PrivateKey)
}

// ParseAccessToken verifies the signature and expiry of a token issued by
// this provider and returns its claims.
func (p *Provider) ParseAccessToken(raw string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return &p.keys.PrivateKey.PublicKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}

func (p *Provider) JWKS() map[string]interface{} {
	return map[string]interface{}{
		"keys": []interface{}{p.keys.JWK()},
	}
}

func (p *Provider) DiscoveryDocument() map[string]interface{} {
	base := p.cfg.Issuer
	return map[string]interface{}{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth2/authorize",
		"token_endpoint":                        base + "/oauth2/token",
		"userinfo_endpoint":                     base + "/oauth2/userinfo",
		"jwks_uri":                              base + "/oauth2/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "roles"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
	}
}

func verifyPKCE(challenge, verifier string) bool {
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
