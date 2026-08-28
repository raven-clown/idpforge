CREATE TABLE users (
	id UNIQUEIDENTIFIER PRIMARY KEY,
	username NVARCHAR(255) NOT NULL UNIQUE,
	email NVARCHAR(320) NOT NULL UNIQUE,
	password_hash NVARCHAR(255),
	status NVARCHAR(32) NOT NULL DEFAULT 'active',
	mfa_enabled BIT NOT NULL DEFAULT 0,
	mfa_secret NVARCHAR(255),
	webauthn_credentials NVARCHAR(MAX) NOT NULL DEFAULT '[]',
	source NVARCHAR(64) NOT NULL DEFAULT 'local',
	external_id NVARCHAR(255),
	created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
	updated_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
	last_login_at DATETIME2 NULL
);

CREATE TABLE groups (
	id UNIQUEIDENTIFIER PRIMARY KEY,
	name NVARCHAR(255) NOT NULL UNIQUE,
	parent_group_id UNIQUEIDENTIFIER NULL REFERENCES groups(id),
	created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
);

CREATE TABLE user_groups (
	user_id UNIQUEIDENTIFIER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	group_id UNIQUEIDENTIFIER NOT NULL REFERENCES groups(id),
	added_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
	PRIMARY KEY (user_id, group_id)
);

CREATE TABLE roles (
	id UNIQUEIDENTIFIER PRIMARY KEY,
	name NVARCHAR(255) NOT NULL UNIQUE,
	description NVARCHAR(MAX),
	created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
);

CREATE TABLE permissions (
	id UNIQUEIDENTIFIER PRIMARY KEY,
	resource NVARCHAR(255) NOT NULL,
	action NVARCHAR(255) NOT NULL,
	CONSTRAINT uq_permissions_resource_action UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
	role_id UNIQUEIDENTIFIER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	permission_id UNIQUEIDENTIFIER NOT NULL REFERENCES permissions(id),
	PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE group_roles (
	group_id UNIQUEIDENTIFIER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
	role_id UNIQUEIDENTIFIER NOT NULL REFERENCES roles(id),
	PRIMARY KEY (group_id, role_id)
);

CREATE TABLE user_roles (
	user_id UNIQUEIDENTIFIER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role_id UNIQUEIDENTIFIER NOT NULL REFERENCES roles(id),
	PRIMARY KEY (user_id, role_id)
);

CREATE TABLE applications (
	id UNIQUEIDENTIFIER PRIMARY KEY,
	name NVARCHAR(255) NOT NULL UNIQUE,
	protocol NVARCHAR(32) NOT NULL,
	config NVARCHAR(MAX) NOT NULL DEFAULT '{}',
	enabled BIT NOT NULL DEFAULT 1,
	created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME(),
	updated_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
);

CREATE INDEX idx_user_groups_group_id ON user_groups(group_id);
CREATE INDEX idx_group_roles_role_id ON group_roles(role_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_groups_parent_group_id ON groups(parent_group_id);

CREATE TABLE audit_logs (
	id BIGINT IDENTITY(1,1) PRIMARY KEY,
	actor_id UNIQUEIDENTIFIER,
	actor_ip NVARCHAR(45),
	actor_user_agent NVARCHAR(MAX),
	action NVARCHAR(255) NOT NULL,
	target_resource NVARCHAR(255),
	target_app NVARCHAR(255),
	before_state NVARCHAR(MAX),
	after_state NVARCHAR(MAX),
	status NVARCHAR(32) NOT NULL,
	trace_id NVARCHAR(64),
	[timestamp] DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
);

CREATE INDEX idx_audit_logs_timestamp ON audit_logs ([timestamp]);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id, [timestamp]);
CREATE INDEX idx_audit_logs_target_app ON audit_logs (target_app, [timestamp]);
