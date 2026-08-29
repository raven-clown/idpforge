ALTER TABLE users ADD COLUMN employee_id VARCHAR(64);
-- MySQL treats multiple NULLs in a UNIQUE index as distinct, so this
-- already allows any number of accounts with no employee_id while still
-- enforcing uniqueness among the ones that have one -- no WHERE clause
-- (which MySQL doesn't support on indexes) needed.
CREATE UNIQUE INDEX idx_users_employee_id ON users (employee_id);
