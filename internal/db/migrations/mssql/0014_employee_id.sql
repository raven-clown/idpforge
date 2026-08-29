ALTER TABLE users ADD employee_id VARCHAR(64);
CREATE UNIQUE NONCLUSTERED INDEX idx_users_employee_id ON users (employee_id) WHERE employee_id IS NOT NULL;
