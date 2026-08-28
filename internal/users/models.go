package users

import "time"

type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDisabled  Status = "disabled"
)

type User struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Status      Status     `json:"status"`
	MFAEnabled  bool       `json:"mfa_enabled"`
	Source      string     `json:"source"`
	ExternalID  string     `json:"external_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type CreateInput struct {
	Username string
	Email    string
	Password string
	Source   string
}

type UpdateInput struct {
	Email  *string
	Status *Status
}
