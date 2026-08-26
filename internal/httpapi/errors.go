package httpapi

import "net/http"

type ErrorResponse struct {
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"requestId,omitempty"`
}
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSONHeader(w)
	write(w, status, ErrorResponse{Error: ErrorBody{code, message}, RequestID: r.Header.Get("X-Request-ID")})
}
