package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"oralclear/internal/domain"
	"oralclear/internal/service"
	"strings"
)

type Handler struct{ svc *service.Service }

func New(s *service.Service) *Handler { return &Handler{svc: s} }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSONHeader(w)
	w.Header().Set("X-Request-ID", choose(r.Header.Get("X-Request-ID"), domain.NewID("req")))
	if r.Method == "POST" && r.URL.Path == "/api/v1/clearance-cases" {
		h.create(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/clearance-cases/") {
		h.caseRoute(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/releases/") {
		h.release(w, r)
		return
	}
	if r.URL.Path == "/api/v1/releases" && r.URL.Query().Get("contentDigest") != "" {
		h.release(w, r)
		return
	}
	if r.URL.Path == "/healthz" {
		write(w, 200, map[string]string{"status": "ok"})
		return
	}
	writeErr(w, 404, domain.ErrNotFound)
}
func choose(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func decode(r *http.Request, v interface{}) error {
	if r.ContentLength > 1<<20 {
		return errors.New("请求体过大")
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		return errors.New("Content-Type 必须为 application/json")
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func decodeOptional(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return nil
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if e != nil || len(strings.TrimSpace(string(b))) == 0 {
		return e
	}
	r.Body = io.NopCloser(strings.NewReader(string(b)))
	return json.Unmarshal(b, v)
}
func write(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, status int, e error) {
	write(w, status, map[string]interface{}{"error": map[string]string{"code": code(e), "message": e.Error()}})
}
func code(e error) string {
	switch {
	case errors.Is(e, domain.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(e, domain.ErrConflict):
		return "VERSION_CONFLICT"
	case errors.Is(e, domain.ErrState):
		return "INVALID_STATE"
	case errors.Is(e, domain.ErrForbidden):
		return "FORBIDDEN"
	default:
		return "INVALID_REQUEST"
	}
}

type createReq struct {
	CaseID         string           `json:"caseId"`
	Title          string           `json:"title"`
	CollectionCode string           `json:"collectionCode"`
	ConsentScope   string           `json:"consentScope"`
	PolicyVersion  string           `json:"policyVersion"`
	Segments       []domain.Segment `json:"segments"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var q createReq
	if e := decode(r, &q); e != nil {
		writeErr(w, 400, e)
		return
	}
	c, e := h.svc.CreateCase(service.CreateInput{CaseID: q.CaseID, Title: q.Title, CollectionCode: q.CollectionCode, ConsentScope: q.ConsentScope, PolicyVersion: q.PolicyVersion, Segments: q.Segments}, r.Header.Get("X-Actor"), r.Header.Get("Idempotency-Key"))
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	write(w, 201, c)
}
func (h *Handler) caseRoute(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) < 4 {
		writeErr(w, 404, domain.ErrNotFound)
		return
	}
	id := parts[3]
	if r.Method == "GET" && len(parts) == 4 {
		c, e := h.svc.GetCase(id)
		if e != nil {
			writeErr(w, status(e), e)
			return
		}
		segs, e := h.svc.Segments(id)
		if e != nil {
			writeErr(w, status(e), e)
			return
		}
		fs, e := h.svc.Findings(id)
		if e != nil {
			writeErr(w, status(e), e)
			return
		}
		reviews, e := h.svc.Reviews(id)
		if e != nil {
			writeErr(w, status(e), e)
			return
		}
		write(w, 200, map[string]interface{}{"case": c, "segments": segs, "findings": fs, "reviews": reviews})
		return
	}
	action := ""
	if len(parts) > 4 {
		action = parts[4]
	}
	switch action {
	case "segments":
		h.segment(w, r, id)
	case "lock":
		h.lock(w, r, id)
	case "scan":
		h.scan(w, r, id)
	case "findings":
		var rest []string
		if len(parts) > 5 {
			rest = parts[5:]
		}
		h.finding(w, r, id, rest)
	case "candidate":
		h.candidate(w, r, id)
	case "reviews":
		h.review(w, r, id)
	case "publish":
		h.publish(w, r, id)
	default:
		writeErr(w, 404, domain.ErrNotFound)
	}
}
func (h *Handler) Segments(id string) ([]domain.Segment, error) { return h.svc.Segments(id) }
func (h *Handler) segment(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeErr(w, 405, domain.ErrState)
		return
	}
	var q domain.Segment
	if e := decode(r, &q); e != nil {
		writeErr(w, 400, e)
		return
	}
	e := h.svc.AddSegment(id, q, version(r))
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	write(w, 201, q)
}
func (h *Handler) lock(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeErr(w, 405, domain.ErrState)
		return
	}
	var q struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	_ = decodeOptional(r, &q)
	ev := version(r)
	if q.ExpectedVersion > 0 {
		ev = q.ExpectedVersion
	}
	e := h.svc.LockPolicy(id, ev, actor(r))
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	c, _ := h.svc.GetCase(id)
	write(w, 200, c)
}
func (h *Handler) scan(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeErr(w, 405, domain.ErrState)
		return
	}
	var q struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	_ = decodeOptional(r, &q)
	ev := version(r)
	if q.ExpectedVersion > 0 {
		ev = q.ExpectedVersion
	}
	fs, e := h.svc.Scan(id, ev, actor(r))
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	write(w, 200, map[string]interface{}{"findings": fs})
}
func (h *Handler) finding(w http.ResponseWriter, r *http.Request, id string, rest []string) {
	if r.Method == "GET" {
		fs, e := h.svc.Findings(id)
		if e != nil {
			writeErr(w, status(e), e)
			return
		}
		write(w, 200, fs)
		return
	}
	if r.Method == "POST" && len(rest) > 0 && rest[0] == "batch" {
		var q struct {
			ExpectedVersion int64                     `json:"expectedVersion"`
			Findings        []service.FindingDecision `json:"findings"`
		}
		if e := decode(r, &q); e != nil {
			writeErr(w, 400, e)
			return
		}
		ev := version(r)
		if q.ExpectedVersion > 0 {
			ev = q.ExpectedVersion
		}
		if e := h.svc.DecideBatch(id, q.Findings, ev, actor(r)); e != nil {
			writeErr(w, status(e), e)
			return
		}
		write(w, 200, map[string]string{"status": "updated"})
		return
	}
	if r.Method != "POST" || len(rest) == 0 {
		writeErr(w, 405, domain.ErrState)
		return
	}
	var q struct {
		ExpectedVersion int64  `json:"expectedVersion"`
		Decision        string `json:"decision"`
		ReplacementText string `json:"replacementText"`
		Rationale       string `json:"rationale"`
	}
	if e := decode(r, &q); e != nil {
		writeErr(w, 400, e)
		return
	}
	ev := version(r)
	if q.ExpectedVersion > 0 {
		ev = q.ExpectedVersion
	}
	e := h.svc.Decide(id, rest[0], q.Decision, q.ReplacementText, q.Rationale, ev, actor(r))
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	write(w, 200, map[string]string{"status": "updated"})
}
func (h *Handler) candidate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == "GET" {
		c, e := h.svc.Candidate(id)
		if e != nil {
			writeErr(w, status(e), e)
			return
		}
		write(w, 200, c)
		return
	}
	if r.Method != "POST" {
		writeErr(w, 405, domain.ErrState)
		return
	}
	var q struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	_ = decodeOptional(r, &q)
	ev := version(r)
	if q.ExpectedVersion > 0 {
		ev = q.ExpectedVersion
	}
	c, e := h.svc.BuildCandidate(id, ev, actor(r), r.Header.Get("Idempotency-Key"))
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	write(w, 201, c)
}
func (h *Handler) review(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == "GET" {
		reviews, e := h.svc.Reviews(id)
		if e != nil {
			writeErr(w, status(e), e)
			return
		}
		write(w, 200, reviews)
		return
	}
	if r.Method != "POST" {
		writeErr(w, 405, domain.ErrState)
		return
	}
	var q struct {
		domain.Review
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	if e := decode(r, &q); e != nil {
		writeErr(w, 400, e)
		return
	}
	ev := version(r)
	if q.ExpectedVersion > 0 {
		ev = q.ExpectedVersion
	}
	c, e := h.svc.Review(id, q.Review, ev)
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	write(w, 200, c)
}
func (h *Handler) publish(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "POST" {
		writeErr(w, 405, domain.ErrState)
		return
	}
	var q struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	_ = decodeOptional(r, &q)
	ev := version(r)
	if q.ExpectedVersion > 0 {
		ev = q.ExpectedVersion
	}
	x, e := h.svc.Publish(id, ev, actor(r))
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	write(w, 201, x)
}
func (h *Handler) release(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/releases/")
	if id == r.URL.Path {
		id = ""
	}
	if r.Method != "GET" {
		writeErr(w, 405, domain.ErrState)
		return
	}
	x, e := h.svc.GetRelease(id, r.URL.Query().Get("contentDigest"))
	if e != nil {
		writeErr(w, status(e), e)
		return
	}
	write(w, 200, map[string]interface{}{"release": x, "verified": true})
}
func actor(r *http.Request) string { return choose(r.Header.Get("X-Actor"), "anonymous") }
func version(r *http.Request) int64 {
	var q struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	if r.Method == "GET" {
		return 0
	}
	if v := r.Header.Get("X-Expected-Version"); v != "" {
		var n int64
		_, _ = fmt.Sscan(v, &n)
		return n
	}
	return q.ExpectedVersion
}
func status(e error) int {
	switch {
	case errors.Is(e, domain.ErrNotFound):
		return 404
	case errors.Is(e, domain.ErrConflict):
		return 409
	case errors.Is(e, domain.ErrState):
		return 422
	case errors.Is(e, domain.ErrForbidden):
		return 403
	default:
		return 400
	}
}
