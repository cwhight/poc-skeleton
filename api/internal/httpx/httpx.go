// Package httpx holds HTTP helpers shared across handlers: trusted client-IP
// resolution, an SSE writer with NO write deadline, and consistent JSON errors.
//
// If a protected route is ever added, its guard MUST fail closed (deny on error).
package httpx

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"
)

// ClientIP resolves the client IP from the SECOND-TO-LAST X-Forwarded-For hop.
//
// Why second-to-last: on a trusted single-proxy platform (Cloud Run / a load
// balancer) the platform APPENDS the real client IP as the last hop, so the
// last entry is the proxy's view of its peer and the second-to-last is the
// caller the proxy actually saw. The LEFTMOST entry is fully client-controlled
// and therefore spoofable — a limiter keyed on it is trivially bypassed.
//
// Falls back to the last hop (single entry), then RemoteAddr, so a missing or
// short header never produces an empty key.
func ClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		// Drop empties.
		clean := parts[:0]
		for _, p := range parts {
			if p != "" {
				clean = append(clean, p)
			}
		}
		switch n := len(clean); {
		case n >= 2:
			return clean[n-2] // second-to-last (trusted) hop
		case n == 1:
			return clean[0]
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// SSEWriter prepares w for Server-Sent Events: sets the SSE headers and — the
// critical part — DISABLES the server write deadline so a default timeout can
// never kill a long stream mid-flight after headers are already sent.
//
// It returns a send func (JSON-encodes the payload into one SSE "data:" event
// and flushes) and an ok flag (false if the ResponseWriter cannot flush —
// caller should 500 BEFORE writing any body).
//
// CONTRACT — the wire format the web client (web/app/lib/apiClient.ts) parses:
//
//	event: token   data: {"text":"<chunk>"}    // 0..n, append in order
//	event: done    data: {}                      // stream finished cleanly
//	event: error   data: {"error":"<message>"}   // mid-stream failure
//
// The data is ALWAYS a single line of compact JSON. JSON-encoding is the safe
// choice: a raw chunk containing a newline would otherwise split into multiple
// "data:" lines and silently corrupt token boundaries on the client.
func SSEWriter(w http.ResponseWriter) (send func(event string, payload any), ok bool) {
	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		return nil, false
	}

	// NO write deadline on the stream. Best-effort: ignore the error so a
	// platform that doesn't support deadline control still streams fine.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no") // disable proxy buffering of the stream

	send = func(event string, payload any) {
		// Compact JSON via Marshal — guaranteed single line (no embedded raw
		// newlines), so each event is exactly one "data:" line.
		data, err := json.Marshal(payload)
		if err != nil {
			data = []byte("{}")
		}
		var b strings.Builder
		if event != "" {
			b.WriteString("event: ")
			b.WriteString(event)
			b.WriteByte('\n')
		}
		b.WriteString("data: ")
		b.Write(data)
		b.WriteByte('\n')
		b.WriteByte('\n') // blank line terminates the event
		_, _ = w.Write([]byte(b.String()))
		flusher.Flush()
	}
	return send, true
}

// ErrorBody is the consistent JSON error envelope.
type ErrorBody struct {
	Error string `json:"error"`
}

// WriteError writes a JSON error with the given status. retryAfter > 0 adds a
// Retry-After header (used for 429s).
func WriteError(w http.ResponseWriter, status int, msg string, retryAfter time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	if retryAfter > 0 {
		secs := int(retryAfter.Seconds())
		if secs < 1 {
			secs = 1
		}
		w.Header().Set("Retry-After", itoa(secs))
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{Error: msg})
}

// WriteJSON writes v as JSON with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// MaxBytesMiddleware caps the request body at maxBytes to protect cost and
// latency (server-side; the client can lie). Returns a wrapped handler.
func MaxBytesMiddleware(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if maxBytes > 0 && r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}
