# Architecture — POC Skeleton

Sensible defaults, chosen to match what we already run. A new POC should not need to revisit these; it only fills the idea-config.

## Stack

| Layer | Choice | Why |
|-------|--------|-----|
| Web | **Next.js (App Router)** on **Vercel** | Fast static landing + streamed generate UI; zero-config deploys; cheap/free at POC scale. |
| API | **Go** on **Cloud Run** | Streaming (SSE) friendly, scales to zero, one small service. |
| DB | **Neon Postgres** | Serverless, free tier covers a POC; backs rate-limit counters + signal capture. Use the **pooler** endpoint in `DATABASE_URL`. |
| IaC | **Terraform** | All infra + env vars in Terraform, parameterized by `slug`. No click-ops, ever. |
| LLM | **Swappable provider behind an interface, selected by env var** (`LLM_PROVIDER`) | The model you launch on is rarely the one you keep. No SDK hard-wired across the codebase; user-facing copy stays vendor-neutral. |
| Analytics | Lightweight event sink (platform-native + the 4 funnel events) | Free, no-lock-in; the funnel is the experiment result. |

## Folder layout

```
poc-skeleton/
├── idea.config.example.ts      # THE artifact a new POC fills in
├── configs/                    # worked example configs
│   ├── scopelock.config.ts
│   └── reviewpilot.config.ts
├── web/                        # Next.js (Vercel)
│   └── app/
│       ├── page.tsx            # landing — renders headline/subcopy from config
│       ├── generate/page.tsx   # paste → stream → output + signal
│       ├── api/                # thin route(s): proxy/health if needed
│       └── components/         # OutputView (escape+markdown-safe), SignalCapture, Funnel hooks
├── api/                        # Go (Cloud Run)
│   ├── cmd/                    # main: wire router, deps, server
│   ├── internal/
│   │   ├── generate/           # /generate SSE handler — validate, rate-limit, stream
│   │   ├── llm/                # Provider interface + impls, selected by LLM_PROVIDER
│   │   ├── ratelimit/          # DB-backed per-IP/per-key limiter (shared store)
│   │   ├── store/              # Postgres: signals + rate-limit counters
│   │   └── httpx/              # XFF client-IP, SSE writer (no write deadline), errors
│   └── migrations/             # SQL: signals table, rate_limit table
├── infra/                      # Terraform, parameterized by var.slug
└── .github/workflows/          # CI: test-gate (required) + deploy-from-main
```

## Request flow

```
Browser (web/generate)
  │  1. emit poc_visit on landing
  │  2. user pastes input, clicks generate → emit poc_generate
  ▼
POST /generate  (Go API on Cloud Run)
  │  a. server-side validate: non-empty, within MAX_INPUT_BYTES
  │  b. resolve client IP (trusted XFF hop), DB-backed rate-limit check  → 429 if over
  │  c. build messages from config.systemPrompt + user input
  │  d. llm.Provider.Stream(...)  (provider chosen by LLM_PROVIDER env)
  │  e. stream tokens back as SSE  (NO server write deadline on the stream)
  ▼
Browser renders streamed output
  │  3. user copies/downloads → emit poc_output_action
  │  4. SignalCapture: email or thumbs + "would you use this?" → POST /signal → store → emit poc_signal
  ▼
Postgres: signals row  |  Analytics: funnel events
```

**Rate limiting counts only calls that cost money** (the LLM call), keyed on the trusted IP hop, backed by Postgres so the cap holds across Cloud Run instances and survives deploys. Auth is **not** used in the core POC flow (anonymous-first); if any admin/protected route is added later it must **fail closed**.

## Env vars (all set via Terraform)

| Var | Purpose |
|-----|---------|
| `LLM_PROVIDER` | Selects the provider impl (e.g. `gemini`, `openai`, `anthropic`). |
| `LLM_API_KEY` | Key for the selected provider (only that provider is billed). |
| `LLM_MODEL` | Model id for the selected provider. |
| `DATABASE_URL` | Neon **pooler** connection string. |
| `MAX_INPUT_BYTES` | Server-side input size cap. |
| `RATE_LIMIT_PER_IP` / `RATE_LIMIT_WINDOW` | Abuse cap on the paid endpoint. |
| `POC_SLUG` | Namespaces service, DB, analytics — matches `idea.config.slug`. |
