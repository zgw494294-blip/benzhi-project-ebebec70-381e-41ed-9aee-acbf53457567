package store

import (
	"encoding/json"
	"oralclear/internal/domain"
)

type CaseSnapshot struct {
	Case      *domain.ClearanceCase `json:"case"`
	Segments  []domain.Segment      `json:"segments"`
	Findings  []domain.Finding      `json:"findings"`
	Candidate *domain.Candidate     `json:"candidate,omitempty"`
	Reviews   []domain.Review       `json:"reviews"`
	Release   *domain.Release       `json:"release,omitempty"`
}

func (s *SQLiteStore) Snapshot(id string) (CaseSnapshot, error) {
	c, e := s.GetCase(id)
	if e != nil {
		return CaseSnapshot{}, e
	}
	ss, e := s.ListSegments(id)
	if e != nil {
		return CaseSnapshot{}, e
	}
	fs, e := s.ListFindings(id)
	if e != nil {
		return CaseSnapshot{}, e
	}
	rs, e := s.ListReviews(id)
	if e != nil {
		return CaseSnapshot{}, e
	}
	out := CaseSnapshot{Case: c, Segments: ss, Findings: fs, Reviews: rs}
	if x, e := s.GetCandidate(id); e == nil {
		out.Candidate = x
	}
	if out.Candidate != nil {
		if x, e := s.FindReleaseByDigest(out.Candidate.ContentDigest); e == nil {
			out.Release = x
		}
	}
	return out, nil
}
func (s *SQLiteStore) SnapshotJSON(id string) ([]byte, error) {
	x, e := s.Snapshot(id)
	if e != nil {
		return nil, e
	}
	return json.Marshal(x)
}
func (s *SQLiteStore) ListCaseIDs() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, e := s.db.Query(`SELECT case_id FROM cases ORDER BY created_at`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
