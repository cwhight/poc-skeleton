package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		remoteAddr string
		want       string
	}{
		{
			name: "multi-hop uses second-to-last (trusted) hop",
			// spoofable client value first; platform appends the real peer last.
			xff:  "1.1.1.1, 203.0.113.5, 70.41.3.18",
			want: "203.0.113.5",
		},
		{
			name: "two hops uses the first (second-to-last)",
			xff:  "203.0.113.5, 70.41.3.18",
			want: "203.0.113.5",
		},
		{
			name: "single hop falls back to that entry",
			xff:  "203.0.113.5",
			want: "203.0.113.5",
		},
		{
			name:       "no xff falls back to RemoteAddr host",
			remoteAddr: "198.51.100.7:54321",
			want:       "198.51.100.7",
		},
		{
			name: "whitespace and empties tolerated",
			xff:  " 1.1.1.1 ,  203.0.113.5 , 70.41.3.18 ",
			want: "203.0.113.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.remoteAddr != "" {
				r.RemoteAddr = tt.remoteAddr
			}
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := ClientIP(r); got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSSEWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	send, ok := SSEWriter(rec)
	if !ok {
		t.Fatal("SSEWriter not ok on httptest recorder")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q", ct)
	}
	// The data payload is ALWAYS JSON-encoded into a single "data:" line.
	send("token", map[string]string{"text": "hello"})
	// A chunk containing a newline must NOT split into two data: lines — JSON
	// encoding keeps it a single line (the newline becomes the \n escape).
	send("token", map[string]string{"text": "line1\nline2"})
	send("done", struct{}{})
	out := rec.Body.String()
	if !strings.Contains(out, "event: token\ndata: {\"text\":\"hello\"}\n\n") {
		t.Errorf("missing JSON-wrapped single-line token event: %q", out)
	}
	if !strings.Contains(out, "data: {\"text\":\"line1\\nline2\"}\n\n") {
		t.Errorf("newline in chunk not JSON-escaped into one data line: %q", out)
	}
	if !strings.Contains(out, "event: done\ndata: {}\n\n") {
		t.Errorf("missing JSON-wrapped done event: %q", out)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusTooManyRequests, "slow down", 60_000_000_000) // 60s
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("code = %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q, want 60", rec.Header().Get("Retry-After"))
	}
	if !strings.Contains(rec.Body.String(), `"error":"slow down"`) {
		t.Errorf("body = %q", rec.Body.String())
	}
}
