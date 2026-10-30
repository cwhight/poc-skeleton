package generate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agent-startup/poc-skeleton/api/internal/config"
	"github.com/agent-startup/poc-skeleton/api/internal/store"
)

type fakeSaver struct {
	saved  *store.Signal
	err    error
	called bool
}

func (f *fakeSaver) SaveSignal(_ context.Context, sig store.Signal) error {
	f.called = true
	if f.err != nil {
		return f.err
	}
	f.saved = &sig
	return nil
}

func signalCfg(kind string) config.Config {
	return config.Config{Slug: "test", Idea: config.Idea{Slug: "test", SignalKind: kind}}
}

func TestSignalHandler(t *testing.T) {
	tests := []struct {
		name       string
		kind       string
		method     string
		body       string
		wantStatus int
		wantSaved  bool
	}{
		{"valid email (no discriminator)", "email", http.MethodPost, `{"email":"a@b.com"}`, http.StatusCreated, true},
		{"valid email (explicit kind)", "email", http.MethodPost, `{"kind":"email","email":"a@b.com"}`, http.StatusCreated, true},
		{"email with inputExcerpt", "email", http.MethodPost, `{"kind":"email","email":"a@b.com","inputExcerpt":"some context"}`, http.StatusCreated, true},
		{"invalid email", "email", http.MethodPost, `{"email":"not-an-email"}`, http.StatusBadRequest, false},
		{"empty email", "email", http.MethodPost, `{"email":""}`, http.StatusBadRequest, false},
		{"valid thumbs up (explicit kind)", "thumbs", http.MethodPost, `{"kind":"thumbs","thumbs":"up"}`, http.StatusCreated, true},
		{"valid thumbs down", "thumbs", http.MethodPost, `{"thumbs":"DOWN"}`, http.StatusCreated, true},
		{"invalid thumbs", "thumbs", http.MethodPost, `{"thumbs":"sideways"}`, http.StatusBadRequest, false},
		{"kind mismatch rejected 400", "email", http.MethodPost, `{"kind":"thumbs","thumbs":"up"}`, http.StatusBadRequest, false},
		{"bad json", "email", http.MethodPost, `nope`, http.StatusBadRequest, false},
		{"wrong method", "email", http.MethodGet, "", http.StatusMethodNotAllowed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeSaver{}
			h := SignalHandler(signalCfg(tt.kind), saver)
			req := httptest.NewRequest(tt.method, "/signal", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			h(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if saver.called != tt.wantSaved {
				t.Errorf("saved = %v, want %v", saver.called, tt.wantSaved)
			}
		})
	}
}

func TestSignalHandler_StoresHashedIPNotRaw(t *testing.T) {
	saver := &fakeSaver{}
	h := SignalHandler(signalCfg("email"), saver)
	req := httptest.NewRequest(http.MethodPost, "/signal", strings.NewReader(`{"email":"a@b.com"}`))
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 203.0.113.5, 70.41.3.18")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"recorded"`) {
		t.Errorf("response body = %q, want {\"status\":\"recorded\"}", rec.Body.String())
	}
	if saver.saved == nil {
		t.Fatal("nothing saved")
	}
	if saver.saved.IPHash == "" || strings.Contains(saver.saved.IPHash, ".") {
		t.Errorf("IPHash = %q, expected a hash not a raw IP", saver.saved.IPHash)
	}
	if saver.saved.Slug != "test" {
		t.Errorf("Slug = %q, want test", saver.saved.Slug)
	}
}
