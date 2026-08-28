package rbac

// Permission is a single (resource, action) grant, e.g. ("nifi", "admin") or
// ("grafana", "viewer").
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
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
		if p.Resource == resource && p.Action == action {
			return true
		}
	}
	return false
}
