// Command api is the POC skeleton's single Go service (Cloud Run).
//
// Wires: config from env, the Postgres pool (Neon pooler), the llm.Provider
// chosen by LLM_PROVIDER, the DB-backed rate limiter, and the HTTP router
// (/generate SSE, /signal, /healthz). Listens on $PORT with graceful shutdown.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agent-startup/poc-skeleton/api/internal/config"
	"github.com/agent-startup/poc-skeleton/api/internal/generate"
	"github.com/agent-startup/poc-skeleton/api/internal/httpx"
	"github.com/agent-startup/poc-skeleton/api/internal/llm"
	"github.com/agent-startup/poc-skeleton/api/internal/ratelimit"
	"github.com/agent-startup/poc-skeleton/api/internal/store"
)

func main() {
	cfg, err := config.Load(config.DefaultIdea)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("store: open: %v", err)
	}
	defer st.Close()

	provider, err := llm.New(cfg.LLMProvider, cfg.LLMAPIKey, cfg.LLMModel)
	if err != nil {
		log.Fatalf("llm: %v", err)
	}

	lim := ratelimit.New(st.Pool(), cfg.RateLimitPerIP, cfg.RateLimitWindow, cfg.Slug)

	mux := http.NewServeMux()
	mux.Handle("/generate", httpx.MaxBytesMiddleware(cfg.MaxInputBytes+1024, generate.Handler(cfg, lim, provider)))
	mux.Handle("/signal", httpx.MaxBytesMiddleware(32*1024, generate.SignalHandler(cfg, st)))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := st.Ping(pingCtx); err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "db unavailable", 0)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
		// NB: no WriteTimeout — the /generate SSE stream must not be killed
		// mid-flight. ReadHeaderTimeout still protects against slow-loris.
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("api: listening on :%s (slug=%s, provider=%s)", cfg.Port, cfg.Slug, cfg.LLMProvider)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("api: serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("api: shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("api: shutdown: %v", err)
	}
}
