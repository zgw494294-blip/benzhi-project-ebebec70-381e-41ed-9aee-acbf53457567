package publish_release_orphan

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"oralclear/internal/domain"
	"oralclear/internal/httpapi"
	"oralclear/internal/service"
	"oralclear/internal/store"
	"strings"
	"testing"
)

func TestPublishConflictMustNotPersistRelease(t *testing.T) {
	dsn := "file:publish-release-orphan?mode=memory&cache=shared"
	db, err := store.New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	admin, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()

	caseID := "case-publish-orphan"
	text := "这是一份已批准的公开稿"
	digestBytes := sha256.Sum256([]byte(text))
	digest := hex.EncodeToString(digestBytes[:])
	_, err = admin.Exec(`
		INSERT INTO cases(case_id,title,collection_code,consent_scope,policy_version,status,version,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?)`, caseID, "发布冲突", "COL-1", "公开", "v1", domain.Approved, 5, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	_, err = admin.Exec(`INSERT INTO candidates(case_id,candidate_version,published_text,content_digest,changes_json) VALUES(?,?,?,?,?)`, caseID, 4, text, digest, "[]")
	if err != nil {
		t.Fatal(err)
	}
	// 模拟另一个写入者在发布版本落库后推进案件版本，使条件状态更新确定性冲突。
	_, err = admin.Exec(`CREATE TRIGGER bump_publish_version AFTER INSERT ON releases BEGIN UPDATE cases SET version=version+1 WHERE case_id=NEW.case_id; END`)
	if err != nil {
		t.Fatal(err)
	}

	h := httpapi.RouteHandler(httpapi.New(service.New(db)))
	server := httptest.NewServer(h)
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/clearance-cases/"+caseID+"/publish", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Expected-Version", "5")
	req.Header.Set("X-Actor", "librarian")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("发布冲突未返回 409：status=%d", resp.StatusCode)
	}

	var releaseID string
	if err = admin.QueryRow(`SELECT release_id FROM releases WHERE case_id=?`, caseID).Scan(&releaseID); err != nil {
		t.Fatal(err)
	}
	getResp, err := server.Client().Get(server.URL + "/api/v1/releases/" + releaseID)
	if err != nil {
		t.Fatal(err)
	}
	getBody, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("发布冲突后仍可查询发布版本：status=%d", getResp.StatusCode)
	}
	if len(getBody) == 0 {
		t.Fatal("发布查询未返回响应")
	}
}
