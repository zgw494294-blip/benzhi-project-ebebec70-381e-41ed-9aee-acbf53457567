package swallowed_case_read_error_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"oralclear/internal/domain"
	"oralclear/internal/httpapi"
	"oralclear/internal/service"
	"strings"
	"testing"
)

// faultRepo keeps the case readable while one related collection fails. The
// embedded interface supplies methods that this GET request does not invoke.
type faultRepo struct {
	domain.Repository
	caseData *domain.ClearanceCase
}

func (r *faultRepo) GetCase(string) (*domain.ClearanceCase, error) {
	copy := *r.caseData
	return &copy, nil
}

func (r *faultRepo) ListSegments(string) ([]domain.Segment, error) {
	return []domain.Segment{{SegmentID: "seg-1", CaseID: r.caseData.CaseID, Sequence: 1, SpeakerLabel: "讲述者", OriginalText: "可公开内容", ReviewState: "UNREVIEWED"}}, nil
}

func (r *faultRepo) ListFindings(string) ([]domain.Finding, error) {
	return nil, errors.New("存储读取失败")
}

func (r *faultRepo) ListReviews(string) ([]domain.Review, error) {
	return []domain.Review{}, nil
}

func TestCaseReadMustPropagateRelatedCollectionError(t *testing.T) {
	repo := &faultRepo{caseData: &domain.ClearanceCase{CaseID: "case-read-error", Title: "口述史", CollectionCode: "COL", ConsentScope: "公开", PolicyVersion: "p1", Status: domain.Reviewing, Version: 2}}
	h := httpapi.New(service.New(repo))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/clearance-cases/case-read-error", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body, _ := io.ReadAll(rr.Result().Body)
	if rr.Code == http.StatusOK || !strings.Contains(string(body), "存储读取失败") {
		t.Fatalf("查询错误被吞掉：status=%d body=%s", rr.Code, body)
	}
}
