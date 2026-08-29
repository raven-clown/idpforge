package httpserver

import (
	"net/http"
	"testing"
)

func adminSessionCookie(t *testing.T, h *testHarness) *http.Cookie {
	t.Helper()
	resp := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "admin",
		Password: "AdminBoot123!",
	})
	ck := sessionCookieFrom(resp)
	if ck == nil {
		t.Fatal("expected admin login to set a session cookie")
	}
	return ck
}

// TestCreateUserNeverAcceptsAPasswordField locks in the core constraint
// behind the whole default-password mechanism: there is no request field
// that lets a caller set someone else's password. A body that tries to
// smuggle one in is simply ignored -- the account still gets the
// server-configured default.
func TestCreateUserNeverAcceptsAPasswordField(t *testing.T) {
	h := newTestServer(t, nil)
	admin := adminSessionCookie(t, h)

	resp := doJSON(t, h, http.MethodPost, "/api/v1/users", map[string]string{
		"username": "newhire",
		"email":    "newhire@example.com",
		"password": "AttackerChosenPassw0rd!",
	}, admin)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create user status = %d, want 201", resp.StatusCode)
	}

	// The attacker-supplied password must not work...
	loginAttacker := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "newhire",
		Password: "AttackerChosenPassw0rd!",
	})
	if loginAttacker.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login with the smuggled password: status = %d, want 401", loginAttacker.StatusCode)
	}

	// ...only the configured default does, and it must force a change.
	loginDefault := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "newhire",
		Password: h.cfg.DefaultPassword,
	})
	if loginDefault.StatusCode != http.StatusOK {
		t.Fatalf("login with the default password: status = %d, want 200", loginDefault.StatusCode)
	}
}

// TestResetPasswordNeverReturnsAPassword covers the other half of the same
// constraint: resetting a user's password returns the updated User object
// and nothing else -- no endpoint ever hands a password back to a caller.
func TestResetPasswordNeverReturnsAPassword(t *testing.T) {
	h := newTestServer(t, nil)
	admin := adminSessionCookie(t, h)

	createResp := doJSON(t, h, http.MethodPost, "/api/v1/users", map[string]string{
		"username": "resetme",
		"email":    "resetme@example.com",
	}, admin)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create user status = %d, want 201", createResp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(t, createResp, &created); err != nil {
		t.Fatalf("decode created user: %v", err)
	}

	resetResp := doJSON(t, h, http.MethodPost, "/api/v1/users/"+created.ID+"/reset-password", nil, admin)
	if resetResp.StatusCode != http.StatusOK {
		t.Fatalf("reset-password status = %d, want 200", resetResp.StatusCode)
	}

	var body map[string]any
	if err := decodeJSON(t, resetResp, &body); err != nil {
		t.Fatalf("decode reset-password response: %v", err)
	}
	for k := range body {
		if k == "password" || k == "password_hash" {
			t.Fatalf("reset-password response leaked a %q field", k)
		}
	}
	if forced, _ := body["force_password_change"].(bool); !forced {
		t.Error("expected force_password_change=true after a reset")
	}
}
