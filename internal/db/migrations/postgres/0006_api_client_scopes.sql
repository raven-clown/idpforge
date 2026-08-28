-- Scopes are the same "resource:action" grants as the roles/permissions
-- system, e.g. ["users:read","users:manage","rbac:manage"], plus the same
-- "resource/*" wildcard support. Set explicitly by an admin per client,
-- same model as a GitHub personal access token.
ALTER TABLE api_clients ADD COLUMN scopes JSONB NOT NULL DEFAULT '[]';
