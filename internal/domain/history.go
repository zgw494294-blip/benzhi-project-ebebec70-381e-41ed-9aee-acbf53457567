package domain

import "time"

type StatusEvent struct {
	CaseID string
	Status CaseStatus
	Actor  string
	At     time.Time
}

func (e StatusEvent) Valid() bool { return e.CaseID != "" && e.Actor != "" && !e.At.IsZero() }
