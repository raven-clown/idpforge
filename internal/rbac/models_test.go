package rbac

import "testing"

func TestPermissionMatches(t *testing.T) {
	cases := []struct {
		name     string
		perm     Permission
		resource string
		want     bool
	}{
		{"exact match", Permission{Resource: "grafana"}, "grafana", true},
		{"exact mismatch", Permission{Resource: "grafana"}, "vault", false},
		{"global wildcard", Permission{Resource: "*"}, "anything", true},
		{"folder wildcard matches child", Permission{Resource: "reports/*"}, "reports/q1", true},
		{"folder wildcard matches nested child", Permission{Resource: "reports/*"}, "reports/q1/summary", true},
		{"folder wildcard matches the folder itself", Permission{Resource: "reports/*"}, "reports", true},
		{"folder wildcard does not match sibling", Permission{Resource: "reports/*"}, "reportsx", false},
		{"folder wildcard does not match unrelated", Permission{Resource: "reports/*"}, "other", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.perm.matches(tc.resource); got != tc.want {
				t.Errorf("Permission{%q}.matches(%q) = %v, want %v", tc.perm.Resource, tc.resource, got, tc.want)
			}
		})
	}
}

func TestResolvedHas(t *testing.T) {
	resolved := Resolved{
		Permissions: []Permission{
			{Resource: "grafana", Action: "viewer"},
			{Resource: "reports/*", Action: "read"},
		},
	}

	if !resolved.Has("grafana", "viewer") {
		t.Error("expected grafana:viewer to be granted")
	}
	if resolved.Has("grafana", "admin") {
		t.Error("did not expect grafana:admin to be granted")
	}
	if !resolved.Has("reports/q1", "read") {
		t.Error("expected reports/q1:read to be granted via folder wildcard")
	}
	if resolved.Has("reports/q1", "write") {
		t.Error("did not expect reports/q1:write to be granted (wrong action)")
	}
}
