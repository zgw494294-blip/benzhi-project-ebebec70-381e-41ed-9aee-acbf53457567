package store

import (
	"database/sql"
	"oralclear/internal/domain"
	"time"
)

type AuditEntry struct {
	CaseID, Status, Actor string
	At                    time.Time
}

func (s *SQLiteStore) History(caseID string) ([]AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, e := s.db.Query(`SELECT case_id,status,actor,created_at FROM status_history WHERE case_id=? ORDER BY id`, caseID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var x AuditEntry
		var t string
		if e = rows.Scan(&x.CaseID, &x.Status, &x.Actor, &t); e != nil {
			return nil, e
		}
		x.At, _ = time.Parse(time.RFC3339Nano, t)
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) StatusHistory(caseID string) ([]domain.StatusEvent, error) {
	entries, err := s.History(caseID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.StatusEvent, 0, len(entries))
	for _, e := range entries {
		out = append(out, domain.StatusEvent{CaseID: e.CaseID, Status: domain.CaseStatus(e.Status), Actor: e.Actor, At: e.At})
	}
	return out, nil
}
func (s *SQLiteStore) SchemaVersion() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var n int
	e := s.db.QueryRow(`SELECT version FROM schema_meta LIMIT 1`).Scan(&n)
	if e == sql.ErrNoRows {
		return 0, nil
	}
	return n, e
}
func (s *SQLiteStore) CompactHistory(before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.Exec(`DELETE FROM status_history WHERE created_at<?`, before.Format(time.RFC3339Nano))
	return e
}
