-- 0001_init — POC skeleton initial schema.
-- PII minimal (GDPR — Spanish/AEPD jurisdiction): store ip_hash, never the raw IP.

-- signals: anonymous validation-signal capture.
CREATE TABLE IF NOT EXISTS signals (
    id            BIGSERIAL PRIMARY KEY,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    kind          TEXT NOT NULL CHECK (kind IN ('email', 'thumbs')),
    email         TEXT,
    thumbs        TEXT CHECK (thumbs IS NULL OR thumbs IN ('up', 'down')),
    slug          TEXT NOT NULL,
    ip_hash       TEXT NOT NULL,
    input_excerpt TEXT
);

CREATE INDEX IF NOT EXISTS signals_slug_created_at_idx ON signals (slug, created_at);

-- rate_limit: DB-backed per-IP counters (shared store, holds across instances).
CREATE TABLE IF NOT EXISTS rate_limit (
    ip_hash      TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    count        INTEGER NOT NULL DEFAULT 0,
    slug         TEXT NOT NULL,
    PRIMARY KEY (ip_hash, window_start, slug)
);
