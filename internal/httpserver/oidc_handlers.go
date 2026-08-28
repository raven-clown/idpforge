package httpserver

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/auth/oidc"
)

func (s *Server) handleDiscovery(c *fiber.Ctx) error {
	return c.JSON(s.oidc.DiscoveryDocument())
}

func (s *Server) handleJWKS(c *fiber.Ctx) error {
	return c.JSON(s.oidc.JWKS())
}

func (s *Server) handleAuthorize(c *fiber.Ctx) error {
	sessionID := c.Cookies(sessionCookie)
	userID, ok, err := "", false, error(nil)
	if sessionID != "" {
		userID, ok, err = s.sessions.get(c.Context(), sessionID)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "session lookup failed")
	}
	if !ok {
		loginURL := "/login?" + url.Values{"redirect": {c.OriginalURL()}}.Encode()
		return c.Redirect(loginURL, fiber.StatusFound)
	}

	req := oidc.AuthorizeRequest{
		ClientID:            c.Query("client_id"),
		RedirectURI:         c.Query("redirect_uri"),
		Scope:               c.Query("scope", "openid"),
		State:               c.Query("state"),
		CodeChallenge:       c.Query("code_challenge"),
		CodeChallengeMethod: c.Query("code_challenge_method"),
		UserID:              userID,
	}
	if c.Query("response_type") != "code" {
		return fiber.NewError(fiber.StatusBadRequest, "only response_type=code is supported")
	}

	code, err := s.oidc.Authorize(c.Context(), req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	redirect := req.RedirectURI + "?" + url.Values{"code": {code}, "state": {req.State}}.Encode()
	return c.Redirect(redirect, fiber.StatusFound)
}

func (s *Server) handleToken(c *fiber.Ctx) error {
	grantType := c.FormValue("grant_type")
	clientID, clientSecret := clientCredentials(c)

	switch grantType {
	case "authorization_code":
		resp, err := s.oidc.ExchangeCode(c.Context(), clientID, c.FormValue("code"), c.FormValue("redirect_uri"), c.FormValue("code_verifier"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(resp)
	case "refresh_token":
		resp, err := s.oidc.RefreshTokens(c.Context(), clientID, c.FormValue("refresh_token"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(resp)
	default:
		_ = clientSecret // client_secret verification happens against the applications config once client auth hardening lands
		return fiber.NewError(fiber.StatusBadRequest, "unsupported grant_type")
	}
}

func (s *Server) handleUserinfo(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
	}
	claims, err := s.oidc.ParseAccessToken(strings.TrimPrefix(auth, "Bearer "))
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid access token")
	}

	sub, _ := claims["sub"].(string)
	user, err := s.users.Get(c.Context(), sub)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	return c.JSON(fiber.Map{
		"sub":      user.ID,
		"email":    user.Email,
		"username": user.Username,
		"roles":    claims["roles"],
	})
}

func clientCredentials(c *fiber.Ctx) (id, secret string) {
	if id, secret, ok := basicAuth(c.Get("Authorization")); ok {
		return id, secret
	}
	return c.FormValue("client_id"), c.FormValue("client_secret")
}
