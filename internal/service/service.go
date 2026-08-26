package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"oralclear/internal/domain"
	"sort"
	"strings"
	"time"
)

type Service struct{ repo domain.Repository }

func New(r domain.Repository) *Service                              { return &Service{repo: r} }
func (s *Service) GetCase(id string) (*domain.ClearanceCase, error) { return s.repo.GetCase(id) }

type CreateInput struct {
	CaseID, Title, CollectionCode, ConsentScope, PolicyVersion string
	Segments                                                   []domain.Segment
}

func (s *Service) CreateCase(in CreateInput, actor, key string) (*domain.ClearanceCase, error) {
	if key != "" {
		if v, ok := s.repo.Idempotent("create", key, nil); ok {
			if id, ok := v.(string); ok {
				return s.repo.GetCase(id)
			}
		}
	}
	normalizedScope, err := domain.ValidateConsentScope(in.ConsentScope)
	if err != nil {
		return nil, err
	}
	in.ConsentScope = normalizedScope
	now := time.Now()
	id := in.CaseID
	if id == "" {
		id = domain.NewID("case")
	}
	c := &domain.ClearanceCase{CaseID: id, Title: in.Title, CollectionCode: in.CollectionCode, ConsentScope: in.ConsentScope, PolicyVersion: in.PolicyVersion, Status: domain.Draft, Version: 1, CreatedAt: now, UpdatedAt: now}
	if e := c.Validate(); e != nil {
		return nil, e
	}
	if len(in.Segments) > 0 {
		sort.Slice(in.Segments, func(i, j int) bool { return in.Segments[i].Sequence < in.Segments[j].Sequence })
		for i := range in.Segments {
			if in.Segments[i].Sequence <= 0 || in.Segments[i].OriginalText == "" {
				return nil, domain.ErrInvalid
			}
			in.Segments[i].SegmentID = choose(in.Segments[i].SegmentID, domain.NewID("seg"))
			in.Segments[i].CaseID = id
			in.Segments[i].ReviewState = "UNREVIEWED"
		}
	}
	if e := s.repo.CreateCase(c, actor); e != nil {
		return nil, e
	}
	for i := range in.Segments {
		if e := s.repo.AddSegment(&in.Segments[i]); e != nil {
			return nil, e
		}
	}
	if key != "" {
		_ = s.repo.PutIdempotent("create", key, id)
	}
	return c, nil
}
func choose(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func (s *Service) AddSegment(id string, x domain.Segment, expected int64) error {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return e
	}
	if c.Status != domain.Draft {
		return domain.ErrState
	}
	if expected > 0 && expected != c.Version {
		return domain.ErrConflict
	}
	x.CaseID = id
	x.SegmentID = choose(x.SegmentID, domain.NewID("seg"))
	x.ReviewState = "UNREVIEWED"
	if x.Sequence <= 0 || strings.TrimSpace(x.OriginalText) == "" {
		return domain.ErrInvalid
	}
	return s.repo.AddSegment(&x)
}
func (s *Service) LockPolicy(id string, v int64, actor string) error {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return e
	}
	if c.Status != domain.Draft {
		return domain.ErrState
	}
	if e = domain.ValidateConsent(c.ConsentScope); e != nil {
		return e
	}
	if e = domain.DefaultPolicy(c.PolicyVersion).ValidateConsent(c.ConsentScope); e != nil {
		return e
	}
	return s.repo.UpdateCaseStatus(id, domain.PolicyLocked, v, actor)
}
func (s *Service) Scan(id string, v int64, actor string) ([]domain.Finding, error) {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return nil, e
	}
	if c.Status != domain.PolicyLocked && c.Status != domain.Reviewing {
		return nil, domain.ErrState
	}
	segs, e := s.repo.ListSegments(id)
	if e != nil {
		return nil, e
	}
	if len(segs) == 0 {
		return nil, domain.ErrInvalid
	}
	if e = domain.ValidateSegments(segs); e != nil {
		return nil, e
	}
	key := scanKey(id, c.PolicyVersion, segs)
	if _, ok := s.repo.Idempotent("scan:"+id, key, nil); ok {
		return s.repo.ListFindings(id)
	}
	if e = validateExpected(c.Version, v); e != nil {
		return nil, e
	}
	existing, e := s.repo.ListFindings(id)
	if e != nil {
		return nil, e
	}
	old := make(map[string]domain.Finding)
	for _, f := range existing {
		old[findingTuple(f)] = f
	}
	var reconciled []domain.Finding
	for _, seg := range segs {
		for _, f := range domain.ScanText(seg.SegmentID, seg.OriginalText) {
			if prior, ok := old[findingTuple(f)]; ok {
				f.FindingID, f.Decision, f.ReplacementText, f.Rationale = prior.FindingID, prior.Decision, prior.ReplacementText, prior.Rationale
			}
			reconciled = append(reconciled, f)
		}
	}
	if bw, ok := s.repo.(domain.FindingBatchWriter); ok {
		if e = bw.ReplaceFindings(id, reconciled); e != nil {
			return nil, e
		}
	} else {
		for i := range reconciled {
			if e = s.repo.SaveFinding(&reconciled[i]); e != nil {
				return nil, e
			}
		}
	}
	if c.Status == domain.PolicyLocked {
		if e = s.repo.UpdateCaseStatus(id, domain.Reviewing, v, actor); e != nil {
			return nil, e
		}
	}
	_ = s.repo.PutIdempotent("scan:"+id, key, true)
	return s.repo.ListFindings(id)
}

func findingTuple(f domain.Finding) string {
	return fmt.Sprintf("%s|%s|%d|%d", f.SegmentID, f.Category, f.StartOffset, f.EndOffset)
}

func scanKey(id, policy string, segs []domain.Segment) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s", id, policy)
	for _, seg := range segs {
		_, _ = fmt.Fprintf(h, "|%s|%d|%s", seg.SegmentID, seg.Sequence, seg.OriginalText)
	}
	return hex.EncodeToString(h.Sum(nil))
}
func (s *Service) Findings(id string) ([]domain.Finding, error) {
	if _, e := s.repo.GetCase(id); e != nil {
		return nil, e
	}
	return s.repo.ListFindings(id)
}
func (s *Service) Segments(id string) ([]domain.Segment, error) {
	if _, e := s.repo.GetCase(id); e != nil {
		return nil, e
	}
	return s.repo.ListSegments(id)
}
func (s *Service) Candidate(id string) (*domain.Candidate, error) {
	if _, e := s.repo.GetCase(id); e != nil {
		return nil, e
	}
	return s.repo.GetCandidate(id)
}
func (s *Service) Decide(id, fid, decision, repl, why string, v int64, actor string) error {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return e
	}
	if c.Status != domain.Reviewing {
		return domain.ErrState
	}
	if e = validateExpected(c.Version, v); e != nil {
		return e
	}
	if e = validateActor(actor); e != nil {
		return e
	}
	fs, e := s.repo.ListFindings(id)
	if e != nil {
		return e
	}
	var target *domain.Finding
	for i := range fs {
		if fs[i].FindingID == fid {
			target = &fs[i]
			break
		}
	}
	if target == nil {
		return domain.ErrNotFound
	}
	segText := ""
	if segs, er := s.repo.ListSegments(id); er == nil {
		for _, seg := range segs {
			if seg.SegmentID == target.SegmentID {
				segText = seg.OriginalText
				break
			}
		}
	}
	target.Decision, target.ReplacementText, target.Rationale = decision, repl, why
	if e = domain.ValidateFinding(*target, segText); e != nil {
		return e
	}
	if e = s.repo.UpdateFinding(target); e != nil {
		return e
	}
	_ = actor
	_ = v
	return nil
}
func (s *Service) BuildCandidate(id string, v int64, actor, key string) (*domain.Candidate, error) {
	if key != "" {
		if _, ok := s.repo.Idempotent("candidate:"+id, key, nil); ok {
			return s.repo.GetCandidate(id)
		}
	}
	c, e := s.repo.GetCase(id)
	if e != nil {
		return nil, e
	}
	if c.Status != domain.Reviewing {
		return nil, domain.ErrState
	}
	if e = s.ValidateCase(id); e != nil {
		return nil, e
	}
	fs, e := s.repo.ListFindings(id)
	if e != nil {
		return nil, e
	}
	for _, f := range fs {
		if f.Decision == "PENDING" {
			return nil, fmt.Errorf("仍有未裁定发现项")
		}
	}
	segs, e := s.repo.ListSegments(id)
	if e != nil {
		return nil, e
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i].Sequence < segs[j].Sequence })
	var texts []string
	var changes []string
	for _, seg := range segs {
		text := seg.OriginalText
		local := []domain.Finding{}
		for _, f := range fs {
			if f.SegmentID == seg.SegmentID {
				local = append(local, f)
			}
		}
		sort.Slice(local, func(i, j int) bool { return local[i].StartOffset > local[j].StartOffset })
		for _, f := range local {
			replacement := text[f.StartOffset:f.EndOffset]
			switch f.Decision {
			case "DELETE":
				replacement = ""
			case "GENERALIZE":
				replacement = "[已泛化]"
			case "REPLACE":
				replacement = f.ReplacementText
			case "KEEP":
			}
			text = text[:f.StartOffset] + replacement + text[f.EndOffset:]
			changes = append(changes, f.FindingID+":"+f.Decision)
		}
		texts = append(texts, seg.SpeakerLabel+": "+text)
	}
	published := strings.Join(texts, "\n")
	sum := sha256.Sum256([]byte(published))
	cand := &domain.Candidate{CaseID: id, CandidateVersion: c.Version + 1, PublishedText: published, ContentDigest: hex.EncodeToString(sum[:]), Changes: changes}
	if e = s.repo.SaveCandidate(cand); e != nil {
		return nil, e
	}
	if e = s.repo.UpdateCaseStatus(id, domain.PendingReview, c.Version, actor); e != nil {
		return nil, e
	}
	if key != "" {
		_ = s.repo.PutIdempotent("candidate:"+id, key, cand)
	}
	return cand, nil
}
func (s *Service) Review(id string, r domain.Review, v int64) (*domain.ClearanceCase, error) {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return nil, e
	}
	if c.Status != domain.PendingReview {
		return nil, domain.ErrState
	}
	if strings.TrimSpace(r.Reviewer) == "" || strings.EqualFold(strings.TrimSpace(r.Reviewer), "unknown") {
		return nil, domain.ErrForbidden
	}
	if e = validateExpected(c.Version, v); e != nil {
		return nil, e
	}
	if hist, ok := s.repo.(domain.StatusHistoryReader); ok {
		events, er := hist.StatusHistory(id)
		if er != nil {
			return nil, er
		}
		for _, ev := range events {
			if sameActor(ev.Actor, r.Reviewer) {
				return nil, domain.ErrForbidden
			}
		}
	}
	reviews, e := s.repo.ListReviews(id)
	if e != nil {
		return nil, e
	}
	if e = reviewIndependent(r.Reviewer, reviews); e != nil {
		return nil, e
	}
	if _, e = s.repo.GetCandidate(id); e != nil {
		return nil, e
	}
	if r.Decision != "APPROVE" && r.Decision != "REJECT" {
		return nil, domain.ErrInvalid
	}
	if r.Decision == "REJECT" && strings.TrimSpace(r.Comment) == "" {
		return nil, domain.ValidationError{Field: "comment", Message: "退回原因不能为空"}
	}
	r.CreatedAt = time.Now()
	status := domain.Approved
	if r.Decision == "REJECT" {
		status = domain.Reviewing
	}
	if committer, ok := s.repo.(domain.ReviewCommitter); ok {
		e = committer.CommitReview(id, &r, status, v, "reviewer:"+r.Reviewer)
	} else {
		if e = s.repo.SaveReview(id, &r); e == nil {
			e = s.repo.UpdateCaseStatus(id, status, v, "reviewer:"+r.Reviewer)
		}
	}
	if e != nil {
		return nil, e
	}
	return s.repo.GetCase(id)
}
func (s *Service) Publish(id string, v int64, actor string) (*domain.Release, error) {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return nil, e
	}
	if c.Status != domain.Approved {
		return nil, domain.ErrState
	}
	cand, e := s.repo.GetCandidate(id)
	if e != nil {
		return nil, e
	}
	for _, p := range domain.SensitivePatterns {
		if p.Re.FindStringIndex(cand.PublishedText) != nil {
			return nil, fmt.Errorf("候选稿仍包含敏感信息：%s", p.Category)
		}
	}
	r := &domain.Release{ReleaseID: domain.NewID("release"), CaseID: id, CandidateVersion: cand.CandidateVersion, PublishedText: cand.PublishedText, ContentDigest: cand.ContentDigest, ApprovedBy: actor, PublishedAt: time.Now()}
	if e = s.repo.SaveRelease(r); e != nil {
		return nil, e
	}
	if e = s.repo.UpdateCaseStatus(id, domain.Published, c.Version, actor); e != nil {
		return nil, e
	}
	return r, nil
}
func (s *Service) GetRelease(id, digest string) (*domain.Release, error) {
	var r *domain.Release
	var e error
	if id != "" {
		r, e = s.repo.GetRelease(id)
	} else {
		r, e = s.repo.FindReleaseByDigest(digest)
	}
	if e != nil {
		return nil, e
	}
	if e = s.VerifyRelease(r); e != nil {
		return nil, e
	}
	return r, nil
}
