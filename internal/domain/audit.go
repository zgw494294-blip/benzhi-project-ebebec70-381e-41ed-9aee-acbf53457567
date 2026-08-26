package domain

import (
	"fmt"
	"time"
)

type AuditTrail struct{ Events []StatusEvent }

func (a *AuditTrail) Append(e StatusEvent) error {
	if !e.Valid() {
		return ErrInvalid
	}
	if len(a.Events) > 0 && !CanTransition(a.Events[len(a.Events)-1].Status, e.Status) {
		return fmt.Errorf("非法状态迁移")
	}
	a.Events = append(a.Events, e)
	return nil
}
func (a AuditTrail) Latest() (StatusEvent, bool) {
	if len(a.Events) == 0 {
		return StatusEvent{}, false
	}
	return a.Events[len(a.Events)-1], true
}
func (a AuditTrail) At(status CaseStatus) time.Time {
	for i := len(a.Events) - 1; i >= 0; i-- {
		if a.Events[i].Status == status {
			return a.Events[i].At
		}
	}
	return time.Time{}
}
