//go:build integration

// Integration tests against a real Postgres (the cached postgres:16 image).
// Run with: go test -tags=integration ./internal/store/...
// Skips automatically if docker is unavailable.
package store_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/agent-startup/poc-skeleton/api/internal/ratelimit"
	"github.com/agent-startup/poc-skeleton/api/internal/store"
)

const (
	pgImage     = "postgres:16"
	pgContainer = "poc-skeleton-pgtest"
	pgPort      = "55432"
	dsn         = "postgres://postgres:postgres@127.0.0.1:" + pgPort + "/postgres?sslmode=disable"
)

func TestIntegration(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker daemon not running")
	}

	// Start a throwaway postgres:16 (do NOT pull — relies on cached image).
	_ = exec.Command("docker", "rm", "-f", pgContainer).Run()
	run := exec.Command("docker", "run", "-d", "--rm",
		"--name", pgContainer,
		"-e", "POSTGRES_PASSWORD=postgres",
		"-p", pgPort+":5432",
		pgImage,
	)
	if out, err := run.CombinedOutput(); err != nil {
		t.Skipf("could not start postgres container (image cached?): %v\n%s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", pgContainer).Run() })

	// Wait for readiness.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var st *store.Store
	var err error
	for {
		st, err = store.Open(ctx, dsn)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("postgres never became ready: %v", err)
		case <-time.After(time.Second):
		}
	}
	defer st.Close()

	// Apply the migration.
	mig, err := os.ReadFile("../../migrations/0001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := st.Pool().Exec(ctx, string(mig)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	t.Run("SaveSignal persists", func(t *testing.T) {
		email := "a@b.com"
		if err := st.SaveSignal(ctx, store.Signal{
			Kind: "email", Email: &email, Slug: "test", IPHash: "abc",
		}); err != nil {
			t.Fatalf("SaveSignal: %v", err)
		}
		var n int
		if err := st.Pool().QueryRow(ctx, `SELECT count(*) FROM signals WHERE slug='test'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("signals count = %d, want 1", n)
		}
	})

	t.Run("limiter atomic increment and cap", func(t *testing.T) {
		lim := ratelimit.New(st.Pool(), 3, time.Hour, "test")
		// 3 calls allowed, 4th over the cap.
		for i := 1; i <= 3; i++ {
			ok, _ := lim.Allow(ctx, "iphash-cap")
			if !ok {
				t.Fatalf("call %d should be allowed", i)
			}
		}
		ok, retry := lim.Allow(ctx, "iphash-cap")
		if ok {
			t.Error("4th call should be blocked")
		}
		if retry <= 0 {
			t.Error("blocked call should report retry-after")
		}
	})
}
