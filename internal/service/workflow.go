package service

import (
	"fmt"
	"oralclear/internal/domain"
	"strings"
)

func (s *Service) ValidateCase(id string) error {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return e
	}
	if e = domain.ValidateConsent(c.ConsentScope); e != nil {
		return e
	}
	segments, e := s.repo.ListSegments(id)
	if e != nil {
		return e
	}
	return domain.ValidateSegments(segments)
}
func (s *Service) EnsureReadyForCandidate(id string) error {
	if e := s.ValidateCase(id); e != nil {
		return e
	}
	fs, e := s.repo.ListFindings(id)
	if e != nil {
		return e
	}
	for _, f := range fs {
		if f.Decision == "PENDING" {
			return fmt.Errorf("发现项 %s 尚未裁定", f.FindingID)
		}
	}
	return nil
}
func (s *Service) RenderCandidate(id string) (string, []string, error) {
	segs, e := s.repo.ListSegments(id)
	if e != nil {
		return "", nil, e
	}
	fs, e := s.repo.ListFindings(id)
	if e != nil {
		return "", nil, e
	}
	var blocks []string
	var changes []string
	for _, seg := range segs {
		var local []domain.Finding
		for _, f := range fs {
			if f.SegmentID == seg.SegmentID {
				local = append(local, f)
			}
		}
		text, ids, e := domain.ApplyFindings(seg.OriginalText, local)
		if e != nil {
			return "", nil, e
		}
		blocks = append(blocks, strings.TrimSpace(seg.SpeakerLabel+": "+text))
		changes = append(changes, ids...)
	}
	return strings.Join(blocks, "\n"), changes, nil
}
func (s *Service) VerifyRelease(r *domain.Release) error {
	if !r.IsImmutable() {
		return domain.ErrInvalid
	}
	if digestText(r.PublishedText) != r.ContentDigest {
		return fmt.Errorf("内容摘要不匹配")
	}
	return nil
}
