package domain

import (
	"fmt"
	"sort"
	"strings"
)

func ValidateSegments(segments []Segment) error {
	if len(segments) == 0 {
		return ErrInvalid
	}
	copySeg := append([]Segment(nil), segments...)
	sort.Slice(copySeg, func(i, j int) bool { return copySeg[i].Sequence < copySeg[j].Sequence })
	for i, s := range copySeg {
		if s.Sequence != i+1 {
			return fmt.Errorf("片段序号必须连续")
		}
		if strings.TrimSpace(s.SpeakerLabel) == "" || strings.TrimSpace(s.OriginalText) == "" {
			return ErrInvalid
		}
	}
	return nil
}
func ValidateFinding(f Finding, source string) error {
	if f.StartOffset < 0 || f.EndOffset <= f.StartOffset || f.EndOffset > len(source) {
		return ErrInvalid
	}
	if f.Decision == "PENDING" {
		return nil
	}
	switch f.Decision {
	case "KEEP", "DELETE", "GENERALIZE":
		return nil
	case "REPLACE":
		if strings.TrimSpace(f.ReplacementText) != "" {
			return nil
		}
	}
	return ErrInvalid
}
func BlockingFindings(fs []Finding) []Finding {
	out := make([]Finding, 0)
	for _, f := range fs {
		if f.Decision != "KEEP" {
			out = append(out, f)
		}
	}
	return out
}
