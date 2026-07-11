package httpx

import (
	"encoding/json"
	"net/http"
)

// Problem is an RFC 7807 problem+json error body — the single error shape every
// service returns, so clients can parse failures uniformly.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Error writes an application/problem+json response. Title is derived from the
// status code; detail is a human-readable, caller-supplied message.
func Error(w http.ResponseWriter, status int, detail string) {
	p := Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
