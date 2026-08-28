package rbac

import "strings"

// Permission is a single (resource, action) grant, e.g. ("nifi", "admin") or
// ("grafana", "viewer"). Resource supports a trailing "/*" wildcard
// ("reports/*") so one grant covers a whole category/folder of resources
// instead of naming each one, and a bare "*" grants every resource.
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func (p Permission) matches(resource string) bool {
	if p.Resource == resource || p.Resource == "*" {
		return true
	}
	if prefix, ok := strings.CutSuffix(p.Resource, "/*"); ok {
		return resource == prefix || strings.HasPrefix(resource, prefix+"/")
	}
	return false
}

func (p Permission) String() string {
	return p.Resource + ":" + p.Action
}

// Resolved is the flattened, cached result of walking
// user -> group(s) [+ ancestor groups] -> role(s) -> permission(s),
// plus any role granted directly to the user (user_roles is the
// per-user override path called out in the schema).
type Resolved struct {
	UserID      string       `json:"user_id"`
	RoleNames   []string     `json:"role_names"`
	Permissions []Permission `json:"permissions"`
}

func (r Resolved) Has(resource, action string) bool {
	for _, p := range r.Permissions {
		if p.Action == action && p.matches(resource) {
			return true
		}
	}
	return false
}
