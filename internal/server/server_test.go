package server

import (
	"net/http"
	"testing"
	"time"
)

// TestNewServerSetsTimeouts asserts the transport-layer timeouts are wired
// (LEARN-104). A zero on any field would let a stuck client hold a request
// goroutine (and the DB query behind it) indefinitely — defeating the
// backstop for LEARN-101's per-query timeouts.
func TestNewServerSetsTimeouts(t *testing.T) {
	srv := newServer(http.NewServeMux())
	for _, tc := range []struct {
		name string
		got  time.Duration
	}{
		{"ReadHeaderTimeout", srv.ReadHeaderTimeout},
		{"ReadTimeout", srv.ReadTimeout},
		{"WriteTimeout", srv.WriteTimeout},
		{"IdleTimeout", srv.IdleTimeout},
	} {
		if tc.got <= 0 {
			t.Errorf("%s = %v, want > 0", tc.name, tc.got)
		}
	}
}
