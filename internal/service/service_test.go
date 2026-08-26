package service

import (
	"oralclear/internal/domain"
	"testing"
)

type mem struct {
	c    *domain.ClearanceCase
	seg  []domain.Segment
	fs   []domain.Finding
	cand *domain.Candidate
	rel  *domain.Release
}

func (m *mem) CreateCase(c *domain.ClearanceCase, _ string) error { m.c = c; return nil }
func (m *mem) GetCase(string) (*domain.ClearanceCase, error) {
	if m.c == nil {
		return nil, domain.ErrNotFound
	}
	return m.c, nil
}
func (m *mem) UpdateCaseStatus(_ string, s domain.CaseStatus, v int64, _ string) error {
	if m.c.Version != v {
		return domain.ErrConflict
	}
	m.c.Status = s
	m.c.Version++
	return nil
}
func (m *mem) AddSegment(s *domain.Segment) error            { m.seg = append(m.seg, *s); return nil }
func (m *mem) ListSegments(string) ([]domain.Segment, error) { return m.seg, nil }
func (m *mem) SaveFinding(f *domain.Finding) error           { m.fs = append(m.fs, *f); return nil }
func (m *mem) ListFindings(string) ([]domain.Finding, error) { return m.fs, nil }
func (m *mem) UpdateFinding(f *domain.Finding) error {
	for i := range m.fs {
		if m.fs[i].FindingID == f.FindingID {
			m.fs[i] = *f
		}
	}
	return nil
}
func (m *mem) SaveCandidate(c *domain.Candidate) error                    { m.cand = c; return nil }
func (m *mem) GetCandidate(string) (*domain.Candidate, error)             { return m.cand, nil }
func (m *mem) SaveReview(string, *domain.Review) error                    { return nil }
func (m *mem) ListReviews(string) ([]domain.Review, error)                { return nil, nil }
func (m *mem) SaveRelease(r *domain.Release) error                        { m.rel = r; return nil }
func (m *mem) GetRelease(string) (*domain.Release, error)                 { return m.rel, nil }
func (m *mem) FindReleaseByDigest(string) (*domain.Release, error)        { return m.rel, nil }
func (m *mem) Idempotent(string, string, interface{}) (interface{}, bool) { return nil, false }
func (m *mem) PutIdempotent(string, string, interface{}) error            { return nil }
func (m *mem) Close() error                                               { return nil }
func TestFlow(t *testing.T) {
	m := &mem{}
	s := New(m)
	c, e := s.CreateCase(CreateInput{Title: "t", CollectionCode: "c", ConsentScope: "公开", PolicyVersion: "p", Segments: []domain.Segment{{Sequence: 1, SpeakerLabel: "甲", OriginalText: "我叫张三"}}}, "a", "")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.LockPolicy(c.CaseID, 1, "a"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Scan(c.CaseID, 2, "a"); e != nil {
		t.Fatal(e)
	}
	if len(m.fs) == 0 {
		t.Fatal("未扫描到发现项")
	}
}
