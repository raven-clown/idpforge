package audit

import (
	"fmt"
	"strings"
	"time"

	"context"

	"github.com/raven-clown/idpforge/internal/db"
)

// Record is one row read back from audit_logs, for the admin UI's audit
// log viewer. Before/AfterState are left as raw JSON strings; the caller
// decides whether to render or ignore them.
type Record struct {
	ID             int64     `json:"id"`
	ActorID        string    `json:"actor_id,omitempty"`
	ActorIP        string    `json:"actor_ip,omitempty"`
	ActorUserAgent string    `json:"actor_user_agent,omitempty"`
	Action         string    `json:"action"`
	TargetResource string    `json:"target_resource,omitempty"`
	TargetApp      string    `json:"target_app,omitempty"`
	BeforeState    string    `json:"before_state,omitempty"`
	AfterState     string    `json:"after_state,omitempty"`
	Status         string    `json:"status"`
	TraceID        string    `json:"trace_id,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

type Filter struct {
	ActorID   string
	Action    string
	TargetApp string
	Status    string
	Since     *time.Time
	Until     *time.Time
	Limit     int
	Offset    int
}

type Reader struct {
	db *db.DB
}

func NewReader(database *db.DB) *Reader {
	return &Reader{db: database}
}

func (r *Reader) Query(ctx context.Context, f Filter) ([]Record, error) {
	tsCol := `"timestamp"`
	if r.db.Driver == "mysql" || r.db.Driver == "sqlite" {
		tsCol = "timestamp"
	} else if r.db.Driver == "mssql" {
		tsCol = "[timestamp]"
	}

	where := []string{"1=1"}
	var args []interface{}
	add := func(cond string, val interface{}) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, r.db.Placeholder(len(args))))
	}
	if f.ActorID != "" {
		add("actor_id = %s", f.ActorID)
	}
	if f.Action != "" {
		add("action = %s", f.Action)
	}
	if f.TargetApp != "" {
		add("target_app = %s", f.TargetApp)
	}
	if f.Status != "" {
		add("status = %s", f.Status)
	}
	if f.Since != nil {
		add(tsCol+" >= %s", *f.Since)
	}
	if f.Until != nil {
		add(tsCol+" <= %s", *f.Until)
	}

	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	args = append(args, limit)
	limitPh := r.db.Placeholder(len(args))
	args = append(args, f.Offset)
	offsetPh := r.db.Placeholder(len(args))

	q := fmt.Sprintf(`SELECT id, actor_id, actor_ip, actor_user_agent, action, target_resource, target_app,
before_state, after_state, status, trace_id, %s
FROM audit_logs WHERE %s ORDER BY %s DESC LIMIT %s OFFSET %s`,
		tsCol, strings.Join(where, " AND "), tsCol, limitPh, offsetPh)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Record
	for rows.Next() {
		var rec Record
		var actorID, actorIP, actorUA, targetResource, targetApp, before, after, traceID nullString
		if err := rows.Scan(&rec.ID, &actorID, &actorIP, &actorUA, &rec.Action, &targetResource, &targetApp,
			&before, &after, &rec.Status, &traceID, &rec.Timestamp); err != nil {
			return nil, err
		}
		rec.ActorID = actorID.value
		rec.ActorIP = actorIP.value
		rec.ActorUserAgent = actorUA.value
		rec.TargetResource = targetResource.value
		rec.TargetApp = targetApp.value
		rec.BeforeState = before.value
		rec.AfterState = after.value
		rec.TraceID = traceID.value
		out = append(out, rec)
	}
	return out, rows.Err()
}

// nullString scans a nullable text column into a plain string without
// pulling in database/sql.NullString at every call site.
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
