CREATE TABLE users (
	id CHAR(36) PRIMARY KEY,
	username VARCHAR(255) NOT NULL UNIQUE,
	email VARCHAR(320) NOT NULL UNIQUE,
	password_hash VARCHAR(255),
	status VARCHAR(32) NOT NULL DEFAULT 'active',
	mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
	mfa_secret VARCHAR(255),
	webauthn_credentials JSON NOT NULL,
	source VARCHAR(64) NOT NULL DEFAULT 'local',
	external_id VARCHAR(255),
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	last_login_at TIMESTAMP NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE groups (
	id CHAR(36) PRIMARY KEY,
	name VARCHAR(255) NOT NULL UNIQUE,
	parent_group_id CHAR(36),
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (parent_group_id) REFERENCES groups(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE user_groups (
	user_id CHAR(36) NOT NULL,
	group_id CHAR(36) NOT NULL,
	added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (user_id, group_id),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE roles (
	id CHAR(36) PRIMARY KEY,
	name VARCHAR(255) NOT NULL UNIQUE,
	description TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE permissions (
	id CHAR(36) PRIMARY KEY,
	resource VARCHAR(255) NOT NULL,
	action VARCHAR(255) NOT NULL,
	UNIQUE (resource, action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE role_permissions (
	role_id CHAR(36) NOT NULL,
	permission_id CHAR(36) NOT NULL,
	PRIMARY KEY (role_id, permission_id),
	FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
	FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE group_roles (
	group_id CHAR(36) NOT NULL,
	role_id CHAR(36) NOT NULL,
	PRIMARY KEY (group_id, role_id),
	FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
	FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE user_roles (
	user_id CHAR(36) NOT NULL,
	role_id CHAR(36) NOT NULL,
	PRIMARY KEY (user_id, role_id),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE applications (
	id CHAR(36) PRIMARY KEY,
	name VARCHAR(255) NOT NULL UNIQUE,
	protocol VARCHAR(32) NOT NULL,
	config JSON NOT NULL,
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_user_groups_group_id ON user_groups(group_id);
CREATE INDEX idx_group_roles_role_id ON group_roles(role_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_groups_parent_group_id ON groups(parent_group_id);

-- No native table partitioning here (MySQL RANGE partitioning on a
-- non-integer timestamp column adds real complexity); rely on the
-- (target_app, timestamp) and (actor_id, timestamp) indexes plus
-- scripts/backup.sh archival for retention instead.
CREATE TABLE audit_logs (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	actor_id CHAR(36),
	actor_ip VARCHAR(45),
	actor_user_agent TEXT,
	action VARCHAR(255) NOT NULL,
	target_resource VARCHAR(255),
	target_app VARCHAR(255),
	before_state JSON,
	after_state JSON,
	status VARCHAR(32) NOT NULL,
	trace_id VARCHAR(64),
	`timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_audit_logs_timestamp ON audit_logs (`timestamp`);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id, `timestamp`);
CREATE INDEX idx_audit_logs_target_app ON audit_logs (target_app, `timestamp`);
