package domain

import (
	"errors"
	"strings"
	"time"
)

type CaseStatus string

const (
	Draft         CaseStatus = "DRAFT"
	PolicyLocked  CaseStatus = "POLICY_LOCKED"
	Reviewing     CaseStatus = "REVIEWING"
	PendingReview CaseStatus = "PENDING_REVIEW"
	Approved      CaseStatus = "APPROVED"
	Published     CaseStatus = "PUBLISHED"
)

var (
	ErrNotFound  = errors.New("资源不存在")
	ErrConflict  = errors.New("版本冲突")
	ErrInvalid   = errors.New("请求不符合业务规则")
	ErrState     = errors.New("当前状态不允许此操作")
	ErrForbidden = errors.New("操作人不具备所需独立性")
)

type ClearanceCase struct {
	CaseID         string     `json:"caseId"`
	Title          string     `json:"title"`
	CollectionCode string     `json:"collectionCode"`
	ConsentScope   string     `json:"consentScope"`
	PolicyVersion  string     `json:"policyVersion"`
	Status         CaseStatus `json:"status"`
	Version        int64      `json:"version"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type Segment struct {
	SegmentID    string `json:"segmentId"`
	CaseID       string `json:"caseId"`
	Sequence     int    `json:"sequence"`
	SpeakerLabel string `json:"speakerLabel"`
	OriginalText string `json:"originalText"`
	ReviewState  string `json:"reviewState"`
}

type Finding struct {
	FindingID       string `json:"findingId"`
	SegmentID       string `json:"segmentId"`
	Category        string `json:"category"`
	StartOffset     int    `json:"startOffset"`
	EndOffset       int    `json:"endOffset"`
	Decision        string `json:"decision"`
	ReplacementText string `json:"replacementText"`
	Rationale       string `json:"rationale"`
}

type Candidate struct {
	CaseID           string   `json:"caseId"`
	CandidateVersion int64    `json:"candidateVersion"`
	PublishedText    string   `json:"publishedText"`
	ContentDigest    string   `json:"contentDigest"`
	Changes          []string `json:"changes"`
}

type Review struct {
	Reviewer  string    `json:"reviewer"`
	Decision  string    `json:"decision"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

type Release struct {
	ReleaseID        string    `json:"releaseId"`
	CaseID           string    `json:"caseId"`
	CandidateVersion int64     `json:"candidateVersion"`
	PublishedText    string    `json:"publishedText"`
	ContentDigest    string    `json:"contentDigest"`
	ApprovedBy       string    `json:"approvedBy"`
	PublishedAt      time.Time `json:"publishedAt"`
}

func (c ClearanceCase) Validate() error {
	if strings.TrimSpace(c.CaseID) == "" || strings.TrimSpace(c.Title) == "" || strings.TrimSpace(c.CollectionCode) == "" || strings.TrimSpace(c.ConsentScope) == "" || strings.TrimSpace(c.PolicyVersion) == "" {
		return ErrInvalid
	}
	if c.Status == "" {
		return ErrInvalid
	}
	return nil
}

func CanTransition(from, to CaseStatus) bool {
	switch from {
	case Draft:
		return to == PolicyLocked
	case PolicyLocked:
		return to == Reviewing
	case Reviewing:
		return to == PendingReview
	case PendingReview:
		return to == Reviewing || to == Approved
	case Approved:
		return to == Published
	default:
		return false
	}
}

type Repository interface {
	CreateCase(*ClearanceCase, string) error
	GetCase(string) (*ClearanceCase, error)
	UpdateCaseStatus(string, CaseStatus, int64, string) error
	AddSegment(*Segment) error
	ListSegments(string) ([]Segment, error)
	SaveFinding(*Finding) error
	ListFindings(string) ([]Finding, error)
	UpdateFinding(*Finding) error
	SaveCandidate(*Candidate) error
	GetCandidate(string) (*Candidate, error)
	SaveReview(string, *Review) error
	ListReviews(string) ([]Review, error)
	SaveRelease(*Release) error
	GetRelease(string) (*Release, error)
	FindReleaseByDigest(string) (*Release, error)
	Idempotent(string, string, interface{}) (interface{}, bool)
	PutIdempotent(string, string, interface{}) error
	Close() error
}

type FindingBatchWriter interface {
	ReplaceFindings(string, []Finding) error
}

type ReviewCommitter interface {
	CommitReview(string, *Review, CaseStatus, int64, string) error
}

type StatusHistoryReader interface {
	StatusHistory(string) ([]StatusEvent, error)
}
