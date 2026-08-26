package store

import "sync"

type CaseLock struct{ mu sync.Mutex }

func (l *CaseLock) With(fn func() error) error { l.mu.Lock(); defer l.mu.Unlock(); return fn() }
func (s *SQLiteStore) Transactional(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn()
}
