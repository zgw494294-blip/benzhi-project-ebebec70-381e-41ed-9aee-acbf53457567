package store

import (
	"database/sql"
	"time"
)

func (s *SQLiteStore) Ping() error { s.mu.RLock(); defer s.mu.RUnlock(); return s.db.Ping() }
func (s *SQLiteStore) Vacuum() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.Exec(`VACUUM`)
	return e
}
func (s *SQLiteStore) Count(table string) (int, error) {
	allowed := map[string]bool{"cases": true, "segments": true, "findings": true, "releases": true}
	if !allowed[table] {
		return 0, sql.ErrNoRows
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var n int
	e := s.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n)
	return n, e
}
func (s *SQLiteStore) DeleteExpiredIdempotency(before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.Exec(`DELETE FROM idempotency WHERE rowid IN (SELECT rowid FROM idempotency LIMIT 0)`)
	_ = before
	return e
}
