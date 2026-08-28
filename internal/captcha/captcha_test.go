package captcha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDispatchesByProvider(t *testing.T) {
	cases := []struct {
		provider string
		wantType interface{}
	}{
		{"turnstile", &turnstileVerifier{}},
		{"hcaptcha", &hcaptchaVerifier{}},
		{"recaptcha", &recaptchaVerifier{}},
		{"none", noopVerifier{}},
		{"unknown", noopVerifier{}},
	}

	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			v := New(tc.provider, "secret")
			if got, want := typeName(v), typeName(tc.wantType); got != want {
				t.Errorf("New(%q) = %T, want %T", tc.provider, v, tc.wantType)
			}
		})
	}
}

func typeName(v interface{}) string {
	switch v.(type) {
	case *turnstileVerifier:
		return "*turnstileVerifier"
	case *hcaptchaVerifier:
		return "*hcaptchaVerifier"
	case *recaptchaVerifier:
		return "*recaptchaVerifier"
	case noopVerifier:
		return "noopVerifier"
	default:
		return "unknown"
	}
}

func TestNoopVerifierAlwaysSucceeds(t *testing.T) {
	ok, err := noopVerifier{}.Verify(context.Background(), "", "")
	if err != nil || !ok {
		t.Errorf("noopVerifier.Verify() = %v, %v, want true, nil", ok, err)
	}
}

func TestTurnstileVerifierSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer server.Close()

	orig := turnstileVerifyURL
	turnstileVerifyURL = server.URL
	defer func() { turnstileVerifyURL = orig }()

	v := &turnstileVerifier{secretKey: "secret"}
	ok, err := v.Verify(context.Background(), "token", "1.2.3.4")
	if err != nil || !ok {
		t.Errorf("Verify() = %v, %v, want true, nil", ok, err)
	}
}

func TestTurnstileVerifierFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"success": false})
	}))
	defer server.Close()

	orig := turnstileVerifyURL
	turnstileVerifyURL = server.URL
	defer func() { turnstileVerifyURL = orig }()

	v := &turnstileVerifier{secretKey: "secret"}
	ok, err := v.Verify(context.Background(), "bad-token", "")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if ok {
		t.Error("expected Verify() to return false for a rejected token")
	}
}

func TestRecaptchaV3LowScoreFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "score": 0.1})
	}))
	defer server.Close()

	orig := recaptchaVerifyURL
	recaptchaVerifyURL = server.URL
	defer func() { recaptchaVerifyURL = orig }()

	v := &recaptchaVerifier{secretKey: "secret"}
	ok, err := v.Verify(context.Background(), "token", "")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if ok {
		t.Error("expected a low reCAPTCHA v3 score to fail verification even though success=true")
	}
}

func TestRecaptchaV3HighScorePasses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "score": 0.9})
	}))
	defer server.Close()

	orig := recaptchaVerifyURL
	recaptchaVerifyURL = server.URL
	defer func() { recaptchaVerifyURL = orig }()

	v := &recaptchaVerifier{secretKey: "secret"}
	ok, err := v.Verify(context.Background(), "token", "")
	if err != nil || !ok {
		t.Errorf("Verify() = %v, %v, want true, nil", ok, err)
	}
}
