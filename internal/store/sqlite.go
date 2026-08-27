package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
	"oralclear/internal/domain"
	"sync"
	"time"
)

type SQLiteStore struct {
	db *sql.DB
	mu sync.RWMutex
}

func New(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &SQLiteStore{db: db}
	if err = s.schema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *SQLiteStore) schema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_meta(version INTEGER NOT NULL)`,
		`INSERT INTO schema_meta(version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_meta)`,
		`CREATE TABLE IF NOT EXISTS cases(case_id TEXT PRIMARY KEY,title TEXT,collection_code TEXT,consent_scope TEXT,policy_version TEXT,status TEXT,version INTEGER,created_at TEXT,updated_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS segments(segment_id TEXT PRIMARY KEY,case_id TEXT,sequence INTEGER,speaker_label TEXT,original_text TEXT,review_state TEXT,UNIQUE(case_id,sequence))`,
		`CREATE TABLE IF NOT EXISTS findings(finding_id TEXT PRIMARY KEY,segment_id TEXT,category TEXT,start_offset INTEGER,end_offset INTEGER,decision TEXT,replacement_text TEXT,rationale TEXT)`,
		`CREATE TABLE IF NOT EXISTS candidates(case_id TEXT PRIMARY KEY,candidate_version INTEGER,published_text TEXT,content_digest TEXT,changes_json TEXT)`,
		`CREATE TABLE IF NOT EXISTS reviews(id INTEGER PRIMARY KEY AUTOINCREMENT,case_id TEXT,reviewer TEXT,decision TEXT,comment TEXT,created_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS releases(release_id TEXT PRIMARY KEY,case_id TEXT UNIQUE,candidate_version INTEGER,published_text TEXT,content_digest TEXT UNIQUE,approved_by TEXT,published_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS idempotency(scope TEXT,key TEXT,result_json TEXT,PRIMARY KEY(scope,key))`,
		`CREATE TABLE IF NOT EXISTS status_history(id INTEGER PRIMARY KEY AUTOINCREMENT,case_id TEXT,status TEXT,actor TEXT,created_at TEXT)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
func (s *SQLiteStore) Close() error { return s.db.Close() }
func (s *SQLiteStore) CreateCase(c *domain.ClearanceCase, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	_, e = tx.Exec(`INSERT INTO cases VALUES(?,?,?,?,?,?,?,?,?)`, c.CaseID, c.Title, c.CollectionCode, c.ConsentScope, c.PolicyVersion, c.Status, c.Version, c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano))
	if e == nil {
		_, e = tx.Exec(`INSERT INTO status_history(case_id,status,actor,created_at) VALUES(?,?,?,?)`, c.CaseID, c.Status, actor, time.Now().Format(time.RFC3339Nano))
	}
	if e != nil {
		tx.Rollback()
		return e
	}
	return tx.Commit()
}
func scanCase(r interface{ Scan(...any) error }) (*domain.ClearanceCase, error) {
	c := &domain.ClearanceCase{}
	var cr, up string
	if e := r.Scan(&c.CaseID, &c.Title, &c.CollectionCode, &c.ConsentScope, &c.PolicyVersion, &c.Status, &c.Version, &cr, &up); e != nil {
		return nil, e
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, cr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, up)
	return c, nil
}
func (s *SQLiteStore) GetCase(id string) (*domain.ClearanceCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, e := scanCase(s.db.QueryRow(`SELECT case_id,title,collection_code,consent_scope,policy_version,status,version,created_at,updated_at FROM cases WHERE case_id=?`, id))
	if e == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return c, e
}
func (s *SQLiteStore) UpdateCaseStatus(id string, st domain.CaseStatus, expected int64, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	now := time.Now().Format(time.RFC3339Nano)
	r, e := tx.Exec(`UPDATE cases SET status=?,version=version+1,updated_at=? WHERE case_id=? AND version=?`, st, now, id, expected)
	if e != nil {
		tx.Rollback()
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		tx.Rollback()
		var exists string
		if er := s.db.QueryRow(`SELECT case_id FROM cases WHERE case_id=?`, id).Scan(&exists); er == sql.ErrNoRows {
			return domain.ErrNotFound
		}
		return domain.ErrConflict
	}
	if _, e = tx.Exec(`INSERT INTO status_history(case_id,status,actor,created_at) VALUES(?,?,?,?)`, id, st, actor, now); e != nil {
		tx.Rollback()
		return e
	}
	return tx.Commit()
}
func (s *SQLiteStore) AddSegment(x *domain.Segment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.Exec(`INSERT INTO segments VALUES(?,?,?,?,?,?)`, x.SegmentID, x.CaseID, x.Sequence, x.SpeakerLabel, x.OriginalText, x.ReviewState)
	return e
}
func (s *SQLiteStore) ListSegments(id string) ([]domain.Segment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, e := s.db.Query(`SELECT segment_id,case_id,sequence,speaker_label,original_text,review_state FROM segments WHERE case_id=? ORDER BY sequence`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []domain.Segment
	for rows.Next() {
		var x domain.Segment
		if e = rows.Scan(&x.SegmentID, &x.CaseID, &x.Sequence, &x.SpeakerLabel, &x.OriginalText, &x.ReviewState); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *SQLiteStore) SaveFinding(f *domain.Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.Exec(`INSERT OR REPLACE INTO findings VALUES(?,?,?,?,?,?,?,?)`, f.FindingID, f.SegmentID, f.Category, f.StartOffset, f.EndOffset, f.Decision, f.ReplacementText, f.Rationale)
	return e
}
func (s *SQLiteStore) ListFindings(caseID string) ([]domain.Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, e := s.db.Query(`SELECT f.finding_id,f.segment_id,f.category,f.start_offset,f.end_offset,f.decision,f.replacement_text,f.rationale FROM findings f JOIN segments s ON f.segment_id=s.segment_id WHERE s.case_id=? ORDER BY s.sequence,f.start_offset`, caseID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []domain.Finding
	for rows.Next() {
		var f domain.Finding
		if e = rows.Scan(&f.FindingID, &f.SegmentID, &f.Category, &f.StartOffset, &f.EndOffset, &f.Decision, &f.ReplacementText, &f.Rationale); e != nil {
			return nil, e
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (s *SQLiteStore) UpdateFinding(f *domain.Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, e := s.db.Exec(`UPDATE findings SET decision=?,replacement_text=?,rationale=? WHERE finding_id=?`, f.Decision, f.ReplacementText, f.Rationale, f.FindingID)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) ReplaceFindings(caseID string, findings []domain.Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM findings WHERE segment_id IN (SELECT segment_id FROM segments WHERE case_id=?)`, caseID); err != nil {
		_ = tx.Rollback()
		return err
	}
	for i := range findings {
		f := findings[i]
		if _, err = tx.Exec(`INSERT INTO findings VALUES(?,?,?,?,?,?,?,?)`, f.FindingID, f.SegmentID, f.Category, f.StartOffset, f.EndOffset, f.Decision, f.ReplacementText, f.Rationale); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
func (s *SQLiteStore) SaveCandidate(c *domain.Candidate) error {
	b, _ := json.Marshal(c.Changes)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.Exec(`INSERT OR REPLACE INTO candidates VALUES(?,?,?,?,?)`, c.CaseID, c.CandidateVersion, c.PublishedText, c.ContentDigest, string(b))
	return e
}
func (s *SQLiteStore) GetCandidate(id string) (*domain.Candidate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var c domain.Candidate
	var ch string
	e := s.db.QueryRow(`SELECT case_id,candidate_version,published_text,content_digest,changes_json FROM candidates WHERE case_id=?`, id).Scan(&c.CaseID, &c.CandidateVersion, &c.PublishedText, &c.ContentDigest, &ch)
	if e == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if e == nil {
		_ = json.Unmarshal([]byte(ch), &c.Changes)
	}
	return &c, e
}
func (s *SQLiteStore) SaveReview(id string, r *domain.Review) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.Exec(`INSERT INTO reviews(case_id,reviewer,decision,comment,created_at) VALUES(?,?,?,?,?)`, id, r.Reviewer, r.Decision, r.Comment, r.CreatedAt.Format(time.RFC3339Nano))
	return e
}

func (s *SQLiteStore) CommitReview(id string, r *domain.Review, status domain.CaseStatus, expected int64, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.db.Exec(`INSERT INTO reviews(case_id,reviewer,decision,comment,created_at) VALUES(?,?,?,?,?)`, id, r.Reviewer, r.Decision, r.Comment, r.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339Nano)
	res, err := tx.Exec(`UPDATE cases SET status=?,version=version+1,updated_at=? WHERE case_id=? AND version=? AND status=?`, status, now, id, expected, domain.PendingReview)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_ = tx.Rollback()
		return domain.ErrConflict
	}
	if _, err = tx.Exec(`INSERT INTO status_history(case_id,status,actor,created_at) VALUES(?,?,?,?)`, id, status, actor, now); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
func (s *SQLiteStore) ListReviews(id string) ([]domain.Review, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, e := s.db.Query(`SELECT reviewer,decision,comment,created_at FROM reviews WHERE case_id=? ORDER BY id`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []domain.Review
	for rows.Next() {
		var r domain.Review
		var t string
		if e = rows.Scan(&r.Reviewer, &r.Decision, &r.Comment, &t); e != nil {
			return nil, e
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, t)
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *SQLiteStore) SaveRelease(r *domain.Release) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.Exec(`INSERT INTO releases VALUES(?,?,?,?,?,?,?)`, r.ReleaseID, r.CaseID, r.CandidateVersion, r.PublishedText, r.ContentDigest, r.ApprovedBy, r.PublishedAt.Format(time.RFC3339Nano))
	return e
}
func (s *SQLiteStore) GetRelease(id string) (*domain.Release, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var r domain.Release
	var t string
	e := s.db.QueryRow(`SELECT release_id,case_id,candidate_version,published_text,content_digest,approved_by,published_at FROM releases WHERE release_id=?`, id).Scan(&r.ReleaseID, &r.CaseID, &r.CandidateVersion, &r.PublishedText, &r.ContentDigest, &r.ApprovedBy, &t)
	if e == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	r.PublishedAt, _ = time.Parse(time.RFC3339Nano, t)
	return &r, e
}
func (s *SQLiteStore) FindReleaseByDigest(d string) (*domain.Release, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var r domain.Release
	var t string
	e := s.db.QueryRow(`SELECT release_id,case_id,candidate_version,published_text,content_digest,approved_by,published_at FROM releases WHERE content_digest=?`, d).Scan(&r.ReleaseID, &r.CaseID, &r.CandidateVersion, &r.PublishedText, &r.ContentDigest, &r.ApprovedBy, &t)
	if e == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	r.PublishedAt, _ = time.Parse(time.RFC3339Nano, t)
	return &r, e
}
func (s *SQLiteStore) Idempotent(scope, key string, _ interface{}) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var b string
	e := s.db.QueryRow(`SELECT result_json FROM idempotency WHERE scope=? AND key=?`, scope, key).Scan(&b)
	if e != nil {
		return nil, false
	}
	var v interface{}
	_ = json.Unmarshal([]byte(b), &v)
	return v, true
}
func (s *SQLiteStore) PutIdempotent(scope, key string, v interface{}) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e = s.db.Exec(`INSERT OR IGNORE INTO idempotency VALUES(?,?,?)`, scope, key, string(b))
	return e
}

var _ = context.Background
var _ = fmt.Sprintf
