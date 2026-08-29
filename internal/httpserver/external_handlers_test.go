package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func jsonRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestExternalCreateUserRequiresUsersManageScope(t *testing.T) {
	h := newTestServer(t, nil)
	ctx := t.Context()

	_, apiKey, err := h.srv.apiClients.Create(ctx, "read-only-integration", "", []string{"id", "username"}, nil, nil, 60, 60)
	if err != nil {
		t.Fatalf("create api client: %v", err)
	}

	req := jsonRequest(t, http.MethodPost, "/external/v1/users", map[string]string{
		"username": "provisioned",
		"email":    "provisioned@example.com",
	})
	req.Header.Set("X-API-Key", apiKey)
	resp, err := h.srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (no users:manage scope)", resp.StatusCode)
	}
}

func TestExternalCreateUserSucceedsWithScope(t *testing.T) {
	h := newTestServer(t, nil)
	ctx := t.Context()

	_, apiKey, err := h.srv.apiClients.Create(ctx, "provisioning-integration", "", []string{"id", "username", "email"}, []string{"users:manage"}, nil, 60, 60)
	if err != nil {
		t.Fatalf("create api client: %v", err)
	}

	req := jsonRequest(t, http.MethodPost, "/external/v1/users", map[string]string{
		"username":    "provisioned",
		"email":       "provisioned@example.com",
		"employee_id": "EMP-001",
		"password":    "ShouldBeIgnored123!",
	})
	req.Header.Set("X-API-Key", apiKey)
	resp, err := h.srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}

	var body map[string]any
	if err := decodeJSON(t, resp, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, present := body["employee_id"]; present {
		t.Error("employee_id leaked through despite not being in allowed_fields")
	}
	if _, present := body["password"]; present {
		t.Error("password field leaked through the response")
	}
	if body["username"] != "provisioned" {
		t.Errorf("username = %v, want provisioned", body["username"])
	}

	// The attacker-supplied "password" field must have been ignored: only
	// the server-configured default password works.
	login := doJSON(t, h, http.MethodPost, "/api/v1/login", loginRequest{
		Username: "provisioned",
		Password: h.cfg.DefaultPassword,
	})
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login with default password status = %d, want 200", login.StatusCode)
	}
}
