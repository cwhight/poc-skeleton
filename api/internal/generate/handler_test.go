package generate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agent-startup/poc-skeleton/api/internal/config"
)

// --- fakes ---

type fakeLimiter struct {
	allow      bool
	retryAfter time.Duration
	calls      int
}

func (f *fakeLimiter) Allow(_ context.Context, _ string) (bool, time.Duration) {
	f.calls++
	return f.allow, f.retryAfter
}

type fakeStreamer struct {
	tokens []string
	err    error
	called bool
}

func (f *fakeStreamer) Stream(_ context.Context, _, _ string) (<-chan string, <-chan error) {
	f.called = true
	out := make(chan string)
	errCh := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errCh)
		for _, t := range f.tokens {
			out <- t
		}
		if f.err != nil {
			errCh <- f.err
		}
	}()
	return out, errCh
}

func testCfg() config.Config {
	idea := config.Idea{Slug: "test", SystemPrompt: "sys", MaxInputBytes: 100, SignalKind: "email"}
	return config.Config{Slug: "test", MaxInputBytes: 100, Idea: idea}
}

// --- /generate handler order + validation ---

func TestGenerateHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		allow          bool
		wantStatus     int
		wantLLMCalled  bool
		wantLimitFirst bool // limiter must be consulted before the LLM
	}{
		{
			name:       "empty input rejected 400",
			method:     http.MethodPost,
			body:       `{"input":"  "}`,
			allow:      true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing input field rejected 400",
			method:     http.MethodPost,
			body:       `{}`,
			allow:      true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "oversize input rejected 400",
			method:     http.MethodPost,
			body:       `{"input":"` + strings.Repeat("x", 200) + `"}`,
			allow:      true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json rejected 400",
			method:     http.MethodPost,
			body:       `not json`,
			allow:      true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:          "rate limited returns 429 before LLM",
			method:        http.MethodPost,
			body:          `{"input":"hello"}`,
			allow:         false,
			wantStatus:    http.StatusTooManyRequests,
			wantLLMCalled: false,
		},
		{
			name:       "wrong method 405",
			method:     http.MethodGet,
			body:       "",
			allow:      true,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:          "happy path streams SSE",
			method:        http.MethodPost,
			body:          `{"input":"hello"}`,
			allow:         true,
			wantStatus:    http.StatusOK,
			wantLLMCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lim := &fakeLimiter{allow: tt.allow, retryAfter: time.Minute}
			str := &fakeStreamer{tokens: []string{"a", "b"}}
			h := Handler(testCfg(), lim, str)

			req := httptest.NewRequest(tt.method, "/generate", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			h(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if str.called != tt.wantLLMCalled {
				t.Errorf("LLM called = %v, want %v", str.called, tt.wantLLMCalled)
			}

			// 429 must carry Retry-After and be JSON (before any SSE headers).
			if tt.wantStatus == http.StatusTooManyRequests {
				if str.called {
					t.Error("LLM was called despite rate limit — must check limit BEFORE the paid call")
				}
				if rec.Header().Get("Retry-After") == "" {
					t.Error("429 missing Retry-After header")
				}
				if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
					t.Errorf("429 Content-Type = %q, want application/json", ct)
				}
			}

			// Happy path: SSE content type + streamed tokens.
			if tt.wantStatus == http.StatusOK {
				if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
					t.Errorf("Content-Type = %q, want text/event-stream", ct)
				}
				out := rec.Body.String()
				// Tokens are JSON-wrapped: event: token / data: {"text":"<chunk>"}.
				if !strings.Contains(out, `event: token`+"\n"+`data: {"text":"a"}`) ||
					!strings.Contains(out, `event: token`+"\n"+`data: {"text":"b"}`) {
					t.Errorf("stream missing JSON-wrapped tokens: %q", out)
				}
				// Terminal frame: event: done / data: {}.
				if !strings.Contains(out, "event: done\ndata: {}\n\n") {
					t.Errorf("stream missing JSON done frame: %q", out)
				}
			}
		})
	}
}

// The limiter must always be consulted (even for invalid input that arrives
// after — here we assert it is called before validation per the baked-in order).
func TestGenerateHandler_LimitCheckedBeforeValidation(t *testing.T) {
	lim := &fakeLimiter{allow: false, retryAfter: time.Minute}
	str := &fakeStreamer{}
	h := Handler(testCfg(), lim, str)

	// Even an oversize/invalid body should be rejected with 429 first, because
	// the limit is checked before reading/validating the body.
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(`garbage`))
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 (limit must be checked before validation)", rec.Code)
	}
	if lim.calls != 1 {
		t.Errorf("limiter calls = %d, want 1", lim.calls)
	}
}
