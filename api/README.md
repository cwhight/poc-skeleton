# api/ — Go (Cloud Run) — STUB

Build agent fills these. One small service: `/generate` (SSE), `/signal`, `/healthz`.

Layout:
- `cmd/main.go` — wire config, DB, provider, router.
- `internal/generate/` — `/generate` SSE handler (+ tests, a hard CI gate) and `/signal`.
- `internal/llm/` — vendor-neutral `Provider` interface; impl chosen by `LLM_PROVIDER`.
- `internal/ratelimit/` — DB-backed per-IP limiter (shared store, not memory).
- `internal/store/` — Postgres (Neon pooler): signals + counters.
- `internal/httpx/` — trusted-XFF client IP, SSE writer (no write deadline), errors.
- `migrations/` — `signals` + `rate_limit` tables.

## HTTP contract (canonical — web/ must match)

- `POST /generate` body `{ "input": string }`. Response `text/event-stream`:
  - `event: token` / `data: {"text":"<chunk>"}` (0..n, append in order)
  - `event: done`  / `data: {}` (clean finish)
  - `event: error` / `data: {"error":"<message>"}` (mid-stream failure)
  - SSE `data` is ALWAYS one line of compact JSON (newline-safe by construction).
  - `429` JSON `{ "error":"…" }` + `Retry-After`, written before any stream bytes.
  - `400` JSON `{ "error":"…" }` on validation (empty / oversize / bad JSON).
- `POST /signal` body `{ "kind":"email", "email" }` or
  `{ "kind":"thumbs", "thumbs":"up"|"down" }` (optional `inputExcerpt`) →
  `201` JSON `{ "status":"recorded" }`; `400` JSON `{ "error":"…" }` on validation.
- `GET /healthz` → `200` `{ "status":"ok" }` (deploy smoke check — do not change).

**Next (per PATTERNS-BAKED-IN):** keep the generate-handler order, the XFF hop, the no-write-deadline SSE, and the DB-backed limiter on the **strong model** (correctness/cost/security-sensitive). Route the provider impls, store boilerplate, migrations, and table-driven tests via the **cheap model**. Go on latest stable. branch → PR → CI green → different-agent review → merge.
