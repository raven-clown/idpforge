package users

import "time"

type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDisabled  Status = "disabled"
)

type User struct {
	ID                  string     `json:"id"`
	Username            string     `json:"username"`
	Email               string     `json:"email"`
	EmployeeID          string     `json:"employee_id,omitempty"`
	Status              Status     `json:"status"`
	MFAEnabled          bool       `json:"mfa_enabled"`
	Source              string     `json:"source"`
	ExternalID          string     `json:"external_id,omitempty"`
	AvatarURL           string     `json:"avatar_url,omitempty"`
	ForcePasswordChange bool       `json:"force_password_change"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
}

type CreateInput struct {
	Username   string
	Email      string
	EmployeeID string
	Password   string
	Source     string
}

type UpdateInput struct {
	Email      *string
	EmployeeID *string
	Status     *Status
}
