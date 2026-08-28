package apiclient

import "testing"

func TestHasScope(t *testing.T) {
	c := Client{Scopes: []string{"users:read", "reports/*:read", "*:health"}}

	cases := []struct {
		name             string
		resource, action string
		want             bool
	}{
		{"exact grant", "users", "read", true},
		{"wrong action", "users", "write", false},
		{"wrong resource", "iot", "read", false},
		{"folder wildcard child", "reports/q1", "read", true},
		{"folder wildcard wrong action", "reports/q1", "write", false},
		{"resource wildcard", "anything", "health", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.HasScope(tc.resource, tc.action); got != tc.want {
				t.Errorf("HasScope(%q, %q) = %v, want %v", tc.resource, tc.action, got, tc.want)
			}
		})
	}
}

func TestHasScopeEmptyGrantsNothing(t *testing.T) {
	c := Client{}
	if c.HasScope("users", "read") {
		t.Error("a client with no scopes should not have any permission")
	}
}
