package generate

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/agent-startup/poc-skeleton/api/internal/config"
	"github.com/agent-startup/poc-skeleton/api/internal/httpx"
	"github.com/agent-startup/poc-skeleton/api/internal/store"
)

// SignalSaver is the store dependency (satisfied by *store.Store).
type SignalSaver interface {
	SaveSignal(ctx context.Context, sig store.Signal) error
}

// signalRequest is the POST /signal body. `kind` is the explicit discriminator
// ("email" | "thumbs"); the matching field (email OR thumbs) carries the value.
// inputExcerpt is optional context (capped server-side).
//
// CONTRACT (matches web/app/lib/apiClient.ts):
//
//	{ "kind": "email",  "email": "<addr>",      "inputExcerpt"?: "<text>" }
//	{ "kind": "thumbs", "thumbs": "up"|"down",  "inputExcerpt"?: "<text>" }
//
// On success: 201 { "status": "recorded" }.
type signalRequest struct {
	Kind         string `json:"kind,omitempty"`
	Email        string `json:"email,omitempty"`
	Thumbs       string `json:"thumbs,omitempty"`
	InputExcerpt string `json:"inputExcerpt,omitempty"`
}

const maxExcerptLen = 280

// SignalHandler records the validation signal — the experiment's payoff event.
// Anonymous (no auth). Validates email format / thumbs enum server-side, then
// persists via SaveSignal.
func SignalHandler(cfg config.Config, saver SignalSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", 0)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 16*1024))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "could not read request body", 0)
			return
		}
		var req signalRequest
		if err := json.Unmarshal(body, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body", 0)
			return
		}

		sig := store.Signal{
			Slug:   cfg.Slug,
			IPHash: hashIP(httpx.ClientIP(r)),
		}
		if ex := strings.TrimSpace(req.InputExcerpt); ex != "" {
			if len(ex) > maxExcerptLen {
				ex = ex[:maxExcerptLen]
			}
			sig.InputExcerpt = &ex
		}

		// Resolve the signal kind. The request carries an explicit `kind`
		// discriminator (the canonical contract); fall back to the config's
		// declared kind if the client omits it. A mismatch is a client error.
		kind := strings.TrimSpace(strings.ToLower(req.Kind))
		if kind == "" {
			kind = cfg.Idea.SignalKind
		} else if kind != cfg.Idea.SignalKind {
			httpx.WriteError(w, http.StatusBadRequest, "unexpected signal kind", 0)
			return
		}

		// Validate per the resolved signal kind.
		switch kind {
		case "email":
			email := strings.TrimSpace(req.Email)
			if _, err := mail.ParseAddress(email); err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "a valid email is required", 0)
				return
			}
			sig.Kind = "email"
			sig.Email = &email
		case "thumbs":
			t := strings.TrimSpace(strings.ToLower(req.Thumbs))
			if t != "up" && t != "down" {
				httpx.WriteError(w, http.StatusBadRequest, "thumbs must be 'up' or 'down'", 0)
				return
			}
			sig.Kind = "thumbs"
			sig.Thumbs = &t
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "signal kind not configured", 0)
			return
		}

		if err := saver.SaveSignal(r.Context(), sig); err != nil {
			log.Printf("signal: save failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "could not record signal", 0)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
	}
}
