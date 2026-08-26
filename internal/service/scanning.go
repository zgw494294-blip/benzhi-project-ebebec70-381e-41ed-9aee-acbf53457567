package service

import (
	"oralclear/internal/domain"
	"sort"
)

func sortFindings(fs []domain.Finding) []domain.Finding {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].SegmentID == fs[j].SegmentID {
			return fs[i].StartOffset < fs[j].StartOffset
		}
		return fs[i].SegmentID < fs[j].SegmentID
	})
	return fs
}
