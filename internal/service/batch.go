package service

import (
	"fmt"
	"oralclear/internal/domain"
	"sort"
	"strings"
)

type FindingDecision struct {
	FindingID       string `json:"findingId"`
	Decision        string `json:"decision"`
	ReplacementText string `json:"replacementText"`
	Rationale       string `json:"rationale"`
}

func (s *Service) AddSegments(id string, segments []domain.Segment, expected int64) error {
	if e := domain.ValidateSegments(segments); e != nil {
		return e
	}
	sort.Slice(segments, func(i, j int) bool { return segments[i].Sequence < segments[j].Sequence })
	for _, x := range segments {
		if e := s.AddSegment(id, x, expected); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) DecideAll(id string, decisions map[string]string, expected int64, actor string) error {
	items := make([]FindingDecision, 0, len(decisions))
	for id, d := range decisions {
		items = append(items, FindingDecision{FindingID: id, Decision: d, Rationale: "批量裁定"})
	}
	return s.DecideBatch(id, items, expected, actor)
}

func (s *Service) DecideBatch(id string, items []FindingDecision, expected int64, actor string) error {
	if len(items) == 0 {
		return domain.ErrInvalid
	}
	c, err := s.repo.GetCase(id)
	if err != nil {
		return err
	}
	if c.Status != domain.Reviewing {
		return domain.ErrState
	}
	if err = validateExpected(c.Version, expected); err != nil {
		return err
	}
	if err = validateActor(actor); err != nil {
		return err
	}
	fs, err := s.repo.ListFindings(id)
	if err != nil {
		return err
	}
	segs, err := s.repo.ListSegments(id)
	if err != nil {
		return err
	}
	segText := make(map[string]string, len(segs))
	for _, seg := range segs {
		segText[seg.SegmentID] = seg.OriginalText
	}
	byID := make(map[string]domain.Finding, len(fs))
	for _, f := range fs {
		byID[f.FindingID] = f
	}
	updated := append([]domain.Finding(nil), fs...)
	idx := make(map[string]int, len(updated))
	for i := range updated {
		idx[updated[i].FindingID] = i
	}
	seen := make(map[string]bool)
	for _, item := range items {
		if strings.TrimSpace(item.FindingID) == "" || seen[item.FindingID] {
			return fmt.Errorf("findingId 无效")
		}
		seen[item.FindingID] = true
		f, ok := byID[item.FindingID]
		if !ok {
			return domain.ErrNotFound
		}
		f.Decision, f.ReplacementText, f.Rationale = item.Decision, item.ReplacementText, item.Rationale
		if err = domain.ValidateFinding(f, segText[f.SegmentID]); err != nil {
			return err
		}
		updated[idx[item.FindingID]] = f
	}
	if bw, ok := s.repo.(domain.FindingBatchWriter); ok {
		return bw.ReplaceFindings(id, updated)
	}
	for i := range updated {
		if updated[i].Decision != fs[i].Decision || updated[i].ReplacementText != fs[i].ReplacementText || updated[i].Rationale != fs[i].Rationale {
			if err = s.repo.UpdateFinding(&updated[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *Service) HasUnresolved(id string) (bool, error) {
	fs, e := s.repo.ListFindings(id)
	if e != nil {
		return false, e
	}
	for _, f := range fs {
		if f.Decision == "PENDING" {
			return true, nil
		}
	}
	return false, nil
}
