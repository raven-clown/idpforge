package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raven-clown/idpforge/internal/config"
	"github.com/raven-clown/idpforge/internal/users"
)

func doJSON(t *testing.T, h *testHarness, method, path string, body any, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", h.cfg.HTTP.BaseURL)
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	resp, err := h.srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, out any) error {
	t.Helper()
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func sessionCookieFrom(resp *http.Response) *http.Cookie {
	for _, ck := range resp.Cookies() {
		if ck.Name == sessionCookie {
			return ck
		}
	}
	return nil
}

func TestLoginSuccess(t *testing.T) {
	h := newTestServer(t, nil)

	resp := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "admin",
		Password: "AdminBoot123!",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}
	if sessionCookieFrom(resp) == nil {
		t.Fatal("expected a session cookie to be set on successful login")
	}

	var out struct {
		PasswordChangeRequired bool `json:"password_change_required"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !out.PasswordChangeRequired {
		t.Error("expected password_change_required=true for the freshly bootstrapped admin account")
	}
}

func TestLoginBadPassword(t *testing.T) {
	h := newTestServer(t, nil)

	resp := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "admin",
		Password: "definitely-wrong",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login status = %d, want 401", resp.StatusCode)
	}
	if sessionCookieFrom(resp) != nil {
		t.Error("expected no session cookie on a failed login")
	}
}

func TestLoginLockoutAfterMaxAttempts(t *testing.T) {
	h := newTestServer(t, func(cfg *config.Config) {
		cfg.AccountLockout = config.AccountLockoutConfig{MaxAttempts: 3, Window: 0, Duration: 0}
	})
	// Window/Duration of 0 still lock (cache.Set with ttl<=0 typically means
	// "no expiry" for the in-memory cache), which is exactly what this test
	// wants: lock and stay locked for the duration of the test.

	for i := 0; i < 3; i++ {
		resp := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
			Username: "admin",
			Password: "wrong",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, resp.StatusCode)
		}
	}

	// A 4th attempt, even with the correct password, must be locked out.
	resp := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "admin",
		Password: "AdminBoot123!",
	})
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("locked-out attempt status = %d, want 429", resp.StatusCode)
	}
}

func TestRequirePermissionRejectsUserWithoutIt(t *testing.T) {
	h := newTestServer(t, nil)
	ctx := context.Background()

	// A plain user with no roles/permissions at all.
	if _, err := h.users.Create(ctx, users.CreateInput{Username: "nobody", Email: "nobody@example.com", Password: "Welcome123!"}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	loginResp := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "nobody",
		Password: "Welcome123!",
	})
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", loginResp.StatusCode)
	}
	ck := sessionCookieFrom(loginResp)
	if ck == nil {
		t.Fatal("expected a session cookie")
	}

	resp := doJSON(t, h, http.MethodGet, "/api/v1/users", nil, ck)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("GET /api/v1/users as a no-permission user: status = %d, want 403", resp.StatusCode)
	}
}

func TestCSRFRejectsMismatchedOrigin(t *testing.T) {
	h := newTestServer(t, nil)

	loginResp := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "admin",
		Password: "AdminBoot123!",
	})
	ck := sessionCookieFrom(loginResp)
	if ck == nil {
		t.Fatal("expected a session cookie")
	}

	var buf bytes.Buffer
	req := httptest.NewRequest(http.MethodPost, "/api/v1/logout", &buf)
	req.Header.Set("Origin", "https://evil.example.com")
	req.AddCookie(ck)
	resp, err := h.srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin cookie-carrying POST: status = %d, want 403", resp.StatusCode)
	}
}
