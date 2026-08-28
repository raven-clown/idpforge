-- A lighter-weight integration path than full OIDC: any internal tool or
-- external service that just needs "verify a login" or "look up a few
-- fields of a user" gets its own key, its own field allowlist, and its own
-- rate limit, instead of implementing the OAuth2 code flow.
CREATE TABLE api_clients (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255) NOT NULL UNIQUE,
	api_key_hash VARCHAR(255) NOT NULL,
	allowed_fields JSONB NOT NULL DEFAULT '["id","username"]',
	rate_limit_max INT NOT NULL DEFAULT 60,
	rate_limit_window_seconds INT NOT NULL DEFAULT 60,
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
