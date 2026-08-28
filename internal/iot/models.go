// Package iot backs physical/embedded integrations: badge readers, face or
// fingerprint terminals, door controllers, canteen kiosks. Matching happens
// on the device itself, same as WebAuthn. This server only sees a
// credential_ref (a card number, or the reader's own template ID), never a
// raw biometric image or template.
package iot

import "time"

type Device struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	DeviceType string    `json:"device_type"`
	Location   string    `json:"location,omitempty"`
	AllowedIPs []string  `json:"allowed_ips,omitempty"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

type Credential struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	CredentialType string    `json:"credential_type"`
	CredentialRef  string    `json:"credential_ref"`
	Label          string    `json:"label,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type Event struct {
	ID             int64     `json:"id"`
	DeviceID       string    `json:"device_id"`
	UserID         string    `json:"user_id,omitempty"`
	CredentialType string    `json:"credential_type,omitempty"`
	CredentialRef  string    `json:"credential_ref,omitempty"`
	EventType      string    `json:"event_type"`
	Resource       string    `json:"resource,omitempty"`
	Metadata       []byte    `json:"metadata,omitempty"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
}

type EventFilter struct {
	UserID    string
	DeviceID  string
	EventType string
	Resource  string
	Since     *time.Time
	Until     *time.Time
	Limit     int
	Offset    int
}

// CredentialProof is one scanned factor. credential_type is free text set by
// the reader ("card", "face_2d", "face_3d", "fingerprint", "iris", ...), so
// higher-fidelity hardware is just a different type string with no
// server-side change needed. Confidence, when reported, is recorded but
// never decides a match; matching is the device's job.
type CredentialProof struct {
	CredentialType string  `json:"credential_type"`
	CredentialRef  string  `json:"credential_ref"`
	Confidence     float64 `json:"confidence,omitempty"`
}

// CheckinRequest normally carries one proof. Send more than one (card +
// face) where single-factor matching risks a false positive, most commonly
// identical twins. All proofs must resolve to the same user or the
// check-in is rejected.
type CheckinRequest struct {
	Credentials []CredentialProof `json:"credentials"`
	EventType   string            `json:"event_type"`
	Resource    string            `json:"resource"`
	Metadata    map[string]any    `json:"metadata"`
}

type CheckinResponse struct {
	Allowed      bool   `json:"allowed"`
	AlreadyToday bool   `json:"already_used_today"`
	UserID       string `json:"user_id,omitempty"`
	EventID      int64  `json:"event_id"`
	Reason       string `json:"reason,omitempty"`
}
