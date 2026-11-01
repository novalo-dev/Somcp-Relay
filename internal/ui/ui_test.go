package ui

import (
	"net/http/httptest"
	"testing"
)

// Without the token, or from another Host, the interface's data and
// actions are refused: another page of the browser cannot drive the relay.
func TestAllowed(t *testing.T) {
	s := &Server{token: "secret", host: "127.0.0.1:5555"}
	ok := httptest.NewRequest("POST", "http://127.0.0.1:5555/api/pause", nil)
	ok.Header.Set("X-Relay-Token", "secret")
	if !s.allowed(ok) {
		t.Error("right token refused")
	}
	noToken := httptest.NewRequest("POST", "http://127.0.0.1:5555/api/pause", nil)
	if s.allowed(noToken) {
		t.Error("no token accepted")
	}
	rebound := httptest.NewRequest("POST", "http://evil.example:5555/api/pause", nil)
	rebound.Header.Set("X-Relay-Token", "secret")
	if s.allowed(rebound) {
		t.Error("foreign Host accepted")
	}
	put := httptest.NewRequest("PUT", "http://127.0.0.1:5555/api/pause", nil)
	put.Header.Set("X-Relay-Token", "secret")
	if s.allowed(put) {
		t.Error("PUT accepted")
	}
}
