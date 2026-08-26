package service

import (
	"fmt"
	"oralclear/internal/domain"
	"sort"
	"strings"
)

type ClearanceReport struct {
	Case                                     domain.ClearanceCase
	SegmentCount, FindingCount, PendingCount int
	Categories                               map[string]int
	Candidate                                *domain.Candidate
	Release                                  *domain.Release
}

func (s *Service) Report(id string) (ClearanceReport, error) {
	var out ClearanceReport
	c, e := s.repo.GetCase(id)
	if e != nil {
		return out, e
	}
	out.Case = *c
	ss, e := s.repo.ListSegments(id)
	if e != nil {
		return out, e
	}
	fs, e := s.repo.ListFindings(id)
	if e != nil {
		return out, e
	}
	out.SegmentCount = len(ss)
	out.FindingCount = len(fs)
	out.Categories = map[string]int{}
	for _, f := range fs {
		out.Categories[f.Category]++
		if f.Decision == "PENDING" {
			out.PendingCount++
		}
	}
	if x, e := s.repo.GetCandidate(id); e == nil {
		out.Candidate = x
	}
	if out.Case.Status == domain.Published && out.Candidate != nil {
		if x, e := s.repo.FindReleaseByDigest(out.Candidate.ContentDigest); e == nil {
			out.Release = x
		}
	}
	return out, nil
}
func FormatReport(r ClearanceReport) string {
	keys := make([]string, 0, len(r.Categories))
	for k := range r.Categories {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "案件 %s 状态 %s\n", r.Case.CaseID, r.Case.Status)
	fmt.Fprintf(&b, "片段 %d，发现项 %d，待处理 %d\n", r.SegmentCount, r.FindingCount, r.PendingCount)
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %d\n", k, r.Categories[k])
	}
	return b.String()
}
func (s *Service) ValidateDigest(id, digest string) error {
	r, e := s.GetRelease("", digest)
	if e != nil {
		return e
	}
	if r.CaseID != id {
		return domain.ErrNotFound
	}
	return nil
}
