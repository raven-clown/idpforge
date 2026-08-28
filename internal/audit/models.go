package audit

import "time"

// Entry is one append-only audit record. BeforeState/AfterState are raw JSON
// (already-marshaled by the caller) so the writer never needs to know the
// shape of the resource being audited.
type Entry struct {
	ActorID        string
	ActorIP        string
	ActorUserAgent string
	Action         string
	TargetResource string
	TargetApp      string
	BeforeState    []byte
	AfterState     []byte
	Status         string
	TraceID        string
	Timestamp      time.Time
}
