// Package apiclient is a lighter alternative to the OIDC provider: any
// internal web app or external service (including AI/automation tools) that
// just needs "verify this login" or "look up a few fields of this user"
// gets its own API key, its own field allowlist, and its own rate limit,
// without implementing the OAuth2 authorization code flow. Grant it Scopes
// and it can also call the real /api/v1 admin API, same model as a GitHub
// personal access token.
package apiclient

import (
	"strings"
	"time"
)

type Client struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	AllowedFields []string `json:"allowed_fields"`
	// Scopes are "resource:action" grants, same format as the roles/
	// permissions system, including a "resource/*" wildcard.
	Scopes []string `json:"scopes,omitempty"`
	// AllowedIPs restricts which source IPs/CIDR ranges may use this key;
	// empty means unrestricted.
	AllowedIPs             []string  `json:"allowed_ips,omitempty"`
	RateLimitMax           int       `json:"rate_limit_max"`
	RateLimitWindowSeconds int       `json:"rate_limit_window_seconds"`
	Enabled                bool      `json:"enabled"`
	CreatedAt              time.Time `json:"created_at"`
}

// HasScope reports whether this client was granted resource:action, with
// the same "resource/*" and bare "*" wildcard rules as RBAC permissions.
func (c Client) HasScope(resource, action string) bool {
	for _, s := range c.Scopes {
		res, act, ok := strings.Cut(s, ":")
		if !ok || act != action {
			continue
		}
		if res == resource || res == "*" {
			return true
		}
		if prefix, cut := strings.CutSuffix(res, "/*"); cut {
			if resource == prefix || strings.HasPrefix(resource, prefix+"/") {
				return true
			}
		}
	}
	return false
}
