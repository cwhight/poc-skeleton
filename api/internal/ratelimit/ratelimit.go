// Package ratelimit is a DB-backed per-identifier rate limiter.
//
// It MUST use the shared store (Postgres), NOT process memory: on autoscaling
// Cloud Run an in-memory limiter enforces N× the cap and resets every deploy.
// A fixed window keyed on (ip_hash, window_start, slug) lets the cap hold across
// instances and survive deploys.
//
// Only the paid /generate call increments the counter, and the handler MUST
// increment (Allow) BEFORE the LLM call so abuse can't run up the bill.
//
// Fail-open: if the DB errors, the request is allowed (logged). A POC must not
// hard-fail the core flow on a transient counter-store hiccup.
package ratelimit

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the subset of pgxpool we need (a *pgxpool.Pool satisfies it). Kept
// as an interface so tests can inject a fake without a live DB.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Limiter increments and checks a per-identifier counter in a fixed window.
type Limiter struct {
	q      Querier
	max    int
	window time.Duration
	slug   string
	now    func() time.Time // injectable for tests
}

// New builds a Limiter from a pgx pool.
func New(pool *pgxpool.Pool, max int, window time.Duration, slug string) *Limiter {
	return &Limiter{q: pool, max: max, window: window, slug: slug, now: time.Now}
}

// NewWithQuerier builds a Limiter over any Querier (used in tests).
func NewWithQuerier(q Querier, max int, window time.Duration, slug string, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{q: q, max: max, window: window, slug: slug, now: now}
}

// Allow atomically increments the counter for identifier in the current window
// and reports whether the request is under the cap. retryAfter is how long
// until the window rolls over (only meaningful when ok == false).
//
// The single statement upserts the row and RETURNs the post-increment count, so
// concurrent instances can't race past the cap (the DB serialises the upsert).
func (l *Limiter) Allow(ctx context.Context, identifier string) (ok bool, retryAfter time.Duration) {
	now := l.now()
	windowStart := now.Truncate(l.window)

	const q = `
		INSERT INTO rate_limit (ip_hash, window_start, slug, count)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (ip_hash, window_start, slug)
		DO UPDATE SET count = rate_limit.count + 1
		RETURNING count`

	var count int
	err := l.q.QueryRow(ctx, q, identifier, windowStart, l.slug).Scan(&count)
	if err != nil {
		// Fail OPEN: never block the core flow on a counter-store hiccup.
		log.Printf("ratelimit: fail-open (db error): %v", err)
		return true, 0
	}

	if count > l.max {
		retry := windowStart.Add(l.window).Sub(now)
		if retry < 0 {
			retry = 0
		}
		return false, retry
	}
	return true, 0
}
