// Package generate holds the core paid endpoint (/generate, SSE) and the
// signal-capture endpoint (/signal).
package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/agent-startup/poc-skeleton/api/internal/config"
	"github.com/agent-startup/poc-skeleton/api/internal/httpx"
)

// Limiter is the rate-limiter dependency (satisfied by *ratelimit.Limiter).
type Limiter interface {
	Allow(ctx context.Context, identifier string) (ok bool, retryAfter time.Duration)
}

// Streamer is the LLM dependency (satisfied by llm.Provider).
type Streamer interface {
	Stream(ctx context.Context, systemPrompt, input string) (<-chan string, <-chan error)
}

// generateRequest is the POST /generate body.
type generateRequest struct {
	Input string `json:"input"`
}

// SSE payload shapes (JSON-encoded by httpx.SSEWriter). These ARE the wire
// contract the web client parses:
//
//	event: token  data: {"text":"<chunk>"}
//	event: error  data: {"error":"<message>"}
//	event: done   data: {}
type tokenPayload struct {
	Text string `json:"text"`
}

type errorPayload struct {
	Error string `json:"error"`
}

// Handler builds the /generate HTTP handler.
//
// ORDER MATTERS — this sequence is a baked-in learning (see PATTERNS-BAKED-IN):
//  1. Resolve client IP from the TRUSTED (second-to-last) XFF hop.
//  2. DB-backed rate-limit check BEFORE the LLM call → real 429 (JSON) if over.
//  3. Server-side validate: non-empty AND within MaxInputBytes.
//  4. Build messages from idea.SystemPrompt + user input.
//  5. Stream tokens as SSE with NO write deadline.
//
// The limit is checked before the (paid) LLM call so abuse can't run up the
// bill, and the 429 is written before any SSE headers go out.
func Handler(cfg config.Config, lim Limiter, provider Streamer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", 0)
			return
		}

		// 1. Trusted client IP, hashed for storage/keying (PII-minimal).
		ip := httpx.ClientIP(r)
		ipHash := hashIP(ip)

		// 2. Rate-limit BEFORE the paid call. Real 429 (JSON) before any headers.
		ok, retryAfter := lim.Allow(r.Context(), ipHash)
		if !ok {
			httpx.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded — try again later", retryAfter)
			return
		}

		// 3. Server-side validation (authoritative — the client can lie).
		//    Body is already capped by MaxBytesMiddleware; enforce non-empty +
		//    size here too in case the handler is mounted without it.
		body, err := io.ReadAll(io.LimitReader(r.Body, cfg.MaxInputBytes+1))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "could not read request body", 0)
			return
		}
		if int64(len(body)) > cfg.MaxInputBytes {
			httpx.WriteError(w, http.StatusBadRequest, "input exceeds maximum size", 0)
			return
		}

		var req generateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body", 0)
			return
		}
		input := strings.TrimSpace(req.Input)
		if input == "" {
			httpx.WriteError(w, http.StatusBadRequest, "input is required", 0)
			return
		}
		if int64(len(input)) > cfg.MaxInputBytes {
			httpx.WriteError(w, http.StatusBadRequest, "input exceeds maximum size", 0)
			return
		}

		// 5. Stream. Open the SSE writer (sets headers + disables write deadline).
		send, canStream := httpx.SSEWriter(w)
		if !canStream {
			httpx.WriteError(w, http.StatusInternalServerError, "streaming unsupported", 0)
			return
		}

		// 4 + 5. Build the prompt from the idea config and stream tokens.
		tokens, errs := provider.Stream(r.Context(), cfg.Idea.SystemPrompt, input)
		for {
			select {
			case <-r.Context().Done():
				return
			case tok, more := <-tokens:
				if !more {
					// Terminal frame: event: done / data: {}
					send("done", struct{}{})
					return
				}
				// Token frame: event: token / data: {"text":"<chunk>"}
				send("token", tokenPayload{Text: tok})
			case err, more := <-errs:
				if !more {
					continue
				}
				if err != nil {
					log.Printf("generate: stream error: %v", err)
					// Error frame: event: error / data: {"error":"<message>"}
					send("error", errorPayload{Error: "generation failed"})
					return
				}
			}
		}
	}
}

// hashIP returns a stable, non-reversible key for an IP (GDPR: store a hash,
// not the raw address). A salt could be added via env for extra unlinkability.
func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}
