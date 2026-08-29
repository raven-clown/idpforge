// Package announcements lets an admin push a message to every signed-in
// user: an update notice, planned downtime, a policy change. Delivered
// live over WebSocket to whoever's connected, and persisted so anyone who
// connects later still sees recent ones.
package announcements

import "time"

type Level string

const (
	LevelInfo     Level = "info"
	LevelWarning  Level = "warning"
	LevelCritical Level = "critical"
)

type Announcement struct {
	ID        int64     `json:"id"`
	Message   string    `json:"message"`
	Level     Level     `json:"level"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
