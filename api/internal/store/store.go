// Package store is Postgres access (Neon pooler). Backs validation signals
// (SaveSignal) and the rate-limit counters (via the *pgxpool.Pool the limiter
// borrows). Use the Neon POOLER endpoint in DATABASE_URL. Migrations live in
// ../migrations. PII is kept minimal (GDPR — Spanish/AEPD): ip_hash, not raw IP.
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// Open parses DATABASE_URL (Neon pooler), opens a small pool, and verifies it.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Pool exposes the underlying pool (the rate limiter borrows it as a shared store).
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// Ping verifies connectivity (used by /healthz).
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Signal is one validation-signal capture. Nullable fields are pointers.
type Signal struct {
	Kind         string  // "email" | "thumbs"
	Email        *string // set when Kind == "email"
	Thumbs       *string // "up" | "down", set when Kind == "thumbs"
	Slug         string
	IPHash       string
	InputExcerpt *string
}

// SaveSignal persists a signal row.
func (s *Store) SaveSignal(ctx context.Context, sig Signal) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO signals (kind, email, thumbs, slug, ip_hash, input_excerpt)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		sig.Kind, sig.Email, sig.Thumbs, sig.Slug, sig.IPHash, sig.InputExcerpt,
	)
	return err
}
