CREATE TABLE users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT,
	status TEXT NOT NULL DEFAULT 'active',
	mfa_enabled INTEGER NOT NULL DEFAULT 0,
	mfa_secret TEXT,
	webauthn_credentials TEXT NOT NULL DEFAULT '[]',
	source TEXT NOT NULL DEFAULT 'local',
	external_id TEXT,
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	last_login_at DATETIME
);

CREATE TABLE groups (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	parent_group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE user_groups (
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
	added_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	PRIMARY KEY (user_id, group_id)
);

CREATE TABLE roles (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT,
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE permissions (
	id TEXT PRIMARY KEY,
	resource TEXT NOT NULL,
	action TEXT NOT NULL,
	UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
	role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
	PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE group_roles (
	group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
	role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	PRIMARY KEY (group_id, role_id)
);

CREATE TABLE user_roles (
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	PRIMARY KEY (user_id, role_id)
);

CREATE TABLE applications (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	protocol TEXT NOT NULL,
	config TEXT NOT NULL DEFAULT '{}',
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX idx_user_groups_group_id ON user_groups(group_id);
CREATE INDEX idx_group_roles_role_id ON group_roles(role_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_groups_parent_group_id ON groups(parent_group_id);

-- Single-node/dev use only; SQLite's single-writer model is not meant for
-- production audit volume. Use Postgres or MySQL for anything multi-user.
CREATE TABLE audit_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	actor_id TEXT,
	actor_ip TEXT,
	actor_user_agent TEXT,
	action TEXT NOT NULL,
	target_resource TEXT,
	target_app TEXT,
	before_state TEXT,
	after_state TEXT,
	status TEXT NOT NULL,
	trace_id TEXT,
	timestamp DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX idx_audit_logs_timestamp ON audit_logs (timestamp);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id, timestamp);
CREATE INDEX idx_audit_logs_target_app ON audit_logs (target_app, timestamp);
