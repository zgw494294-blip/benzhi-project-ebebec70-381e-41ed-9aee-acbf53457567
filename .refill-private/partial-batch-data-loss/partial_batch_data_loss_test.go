package partial_batch_data_loss_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"oralclear/internal/domain"
	"oralclear/internal/httpapi"
	"oralclear/internal/service"
	"oralclear/internal/store"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBatchDecisionFailureMustPreserveFindings(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "oralclear.db")
	repo, err := store.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	svc := service.New(repo)
	c, err := svc.CreateCase(service.CreateInput{
		CaseID:         "case-batch-atomicity",
		Title:          "批量裁定原子性",
		CollectionCode: "OH-ATOMIC",
		ConsentScope:   "公开",
		PolicyVersion:  "p1",
		Segments: []domain.Segment{{
			SegmentID:    "segment-batch-atomicity",
			Sequence:     1,
			SpeakerLabel: "受访者",
			OriginalText: "我叫张三，联系电话13800138000",
		}},
	}, "archivist-a", "")
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.LockPolicy(c.CaseID, 1, "archivist-a"); err != nil {
		t.Fatal(err)
	}
	findings, err := svc.Scan(c.CaseID, 2, "archivist-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("测试前置发现项数量异常：got=%d", len(findings))
	}

	triggerDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = triggerDB.Close() })
	trigger := fmt.Sprintf(`CREATE TRIGGER fail_second_finding BEFORE INSERT ON findings WHEN NEW.finding_id = %q BEGIN SELECT RAISE(ABORT, 'injected finding write failure'); END`, findings[1].FindingID)
	if _, err = triggerDB.Exec(trigger); err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]any{
		"expectedVersion": int64(3),
		"findings": []map[string]string{{
			"findingId": findings[0].FindingID,
			"decision":  "DELETE",
			"rationale": "依授权范围删除姓名",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.New(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clearance-cases/"+c.CaseID+"/findings/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor", "archivist-a")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("注入的存储失败未传回客户端：status=%d", rec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/clearance-cases/"+c.CaseID+"/findings", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("公开查询发现项失败：status=%d", getRec.Code)
	}
	var remaining []domain.Finding
	if err = json.Unmarshal(getRec.Body.Bytes(), &remaining); err != nil {
		t.Fatal(err)
	}
	if len(remaining) != len(findings) {
		t.Fatalf("批量裁定失败后发现项丢失：before=%d after=%d status=%d", len(findings), len(remaining), rec.Code)
	}
}
