package httpserver

import "testing"

func TestOriginOf(t *testing.T) {
	cases := []struct{ in, want string }{
		{"http://localhost:8080", "http://localhost:8080"},
		{"https://sso.example.com", "https://sso.example.com"},
		{"https://SSO.Example.com", "https://sso.example.com"},
		{"https://sso.example.com/oauth2/authorize?x=1", "https://sso.example.com"},
		{"", ""},
		{"not-a-url", ""},
	}
	for _, c := range cases {
		if got := originOf(c.in); got != c.want {
			t.Errorf("originOf(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsStateChanging(t *testing.T) {
	changing := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, m := range changing {
		if !isStateChanging(m) {
			t.Errorf("isStateChanging(%q) = false, want true", m)
		}
	}
	safe := []string{"GET", "HEAD", "OPTIONS"}
	for _, m := range safe {
		if isStateChanging(m) {
			t.Errorf("isStateChanging(%q) = true, want false", m)
		}
	}
}
