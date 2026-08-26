package service

import "oralclear/internal/domain"

type Metrics struct{ Cases, Segments, Findings, Releases int }

func (s *Service) Metrics(id string) (Metrics, error) {
	m := Metrics{}
	if _, e := s.repo.GetCase(id); e != nil {
		return m, e
	}
	ss, e := s.repo.ListSegments(id)
	if e != nil {
		return m, e
	}
	ff, e := s.repo.ListFindings(id)
	if e != nil {
		return m, e
	}
	m.Cases = 1
	m.Segments = len(ss)
	m.Findings = len(ff)
	if r, e := s.repo.FindReleaseByDigest(""); e == nil && r != nil {
		m.Releases = 1
	}
	return m, nil
}

var _ = domain.ErrInvalid
