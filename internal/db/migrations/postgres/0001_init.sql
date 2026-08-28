CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	username VARCHAR(255) NOT NULL UNIQUE,
	email VARCHAR(320) NOT NULL UNIQUE,
	password_hash VARCHAR(255),
	status VARCHAR(32) NOT NULL DEFAULT 'active',
	mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
	mfa_secret VARCHAR(255),
	webauthn_credentials JSONB NOT NULL DEFAULT '[]',
	source VARCHAR(64) NOT NULL DEFAULT 'local',
	external_id VARCHAR(255),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	last_login_at TIMESTAMPTZ
);

CREATE TABLE groups (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255) NOT NULL UNIQUE,
	parent_group_id UUID REFERENCES groups(id) ON DELETE SET NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_groups (
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
	added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (user_id, group_id)
);

CREATE TABLE roles (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255) NOT NULL UNIQUE,
	description TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	resource VARCHAR(255) NOT NULL,
	action VARCHAR(255) NOT NULL,
	UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
	role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
	PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE group_roles (
	group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
	role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	PRIMARY KEY (group_id, role_id)
);

CREATE TABLE user_roles (
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	PRIMARY KEY (user_id, role_id)
);

CREATE TABLE applications (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255) NOT NULL UNIQUE,
	protocol VARCHAR(32) NOT NULL,
	config JSONB NOT NULL DEFAULT '{}',
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_groups_group_id ON user_groups(group_id);
CREATE INDEX idx_group_roles_role_id ON group_roles(role_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_groups_parent_group_id ON groups(parent_group_id);

-- Append-only audit log, partitioned by month. New partitions are created by
-- scripts/create-audit-partition.sh (or the /internal/audit maintenance job)
-- ahead of each month; see docs/runbook.md.
CREATE TABLE audit_logs (
	id BIGINT GENERATED ALWAYS AS IDENTITY,
	actor_id UUID,
	actor_ip INET,
	actor_user_agent TEXT,
	action VARCHAR(255) NOT NULL,
	target_resource VARCHAR(255),
	target_app VARCHAR(255),
	before_state JSONB,
	after_state JSONB,
	status VARCHAR(32) NOT NULL,
	trace_id VARCHAR(64),
	"timestamp" TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (id, "timestamp")
) PARTITION BY RANGE ("timestamp");

CREATE TABLE audit_logs_default PARTITION OF audit_logs DEFAULT;

CREATE INDEX idx_audit_logs_timestamp ON audit_logs ("timestamp");
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id, "timestamp");
CREATE INDEX idx_audit_logs_target_app ON audit_logs (target_app, "timestamp");
