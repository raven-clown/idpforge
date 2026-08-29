package announcements

import (
	"context"
	"fmt"
	"time"

	"github.com/raven-clown/idpforge/internal/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) Create(ctx context.Context, message string, level Level, createdBy string) (*Announcement, error) {
	now := time.Now().UTC()
	q := fmt.Sprintf(`INSERT INTO announcements (message, level, created_by, created_at) VALUES (%s, %s, %s, %s)`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3), r.db.Placeholder(4))

	res, err := r.db.ExecContext(ctx, q, message, string(level), nullableString(createdBy), now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		id = 0 // driver may not support LastInsertId (e.g. pgx); id is informational
	}
	return &Announcement{ID: id, Message: message, Level: level, CreatedBy: createdBy, CreatedAt: now}, nil
}

// List returns the most recent announcements, newest first, for the
// notification bell's initial load (live ones arrive over WebSocket after
// that).
func (r *Repository) List(ctx context.Context, limit int) ([]Announcement, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := fmt.Sprintf(`SELECT id, message, level, created_by, created_at FROM announcements ORDER BY created_at DESC LIMIT %s`,
		r.db.Placeholder(1))

	rows, err := r.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Announcement
	for rows.Next() {
		var a Announcement
		var createdBy nullString
		if err := rows.Scan(&a.ID, &a.Message, &a.Level, &createdBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.CreatedBy = createdBy.value
		out = append(out, a)
	}
	return out, rows.Err()
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

type nullString struct {
	value string
}

func (n *nullString) Scan(src interface{}) error {
	if src == nil {
		n.value = ""
		return nil
	}
	switch v := src.(type) {
	case string:
		n.value = v
	case []byte:
		n.value = string(v)
	default:
		n.value = fmt.Sprintf("%v", v)
	}
	return nil
}
