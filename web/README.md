# web/ — Next.js (App Router) on Vercel

The POC web layer. Everything user-facing reads from `idea.config.ts`, which
defaults to the ScopeLock worked example — **swapping that config = a new POC**.

## Layout

```
web/
├── idea.config.ts            # re-exports the active config (default: scopelock) + types
├── idea.config.types.ts      # IdeaConfig schema (mirror of ../idea.config.example.ts)
├── configs/scopelock.config.ts
├── app/
│   ├── layout.tsx            # metadata/viewport from config
│   ├── globals.css           # constrained palette, >=44px targets, mobile-first
│   ├── page.tsx              # landing — hero from config, single CTA, emits poc_visit
│   ├── generate/page.tsx     # core flow: paste -> SSE stream -> output + signal
│   ├── api/event/route.ts    # thin first-party analytics sink (beacon target)
│   ├── components/
│   │   ├── OutputView.tsx    # SINGLE safe render path: escape-then-markdown
│   │   ├── SignalCapture.tsx # email / thumbs -> POST /signal -> poc_signal
│   │   └── analytics.ts      # the 4 funnel events, slug-namespaced
│   └── lib/apiClient.ts      # /generate SSE consumer + /signal poster
```

## API contract consumed (must match `api/`)

- `POST {NEXT_PUBLIC_API_BASE}/generate` body `{ "input": string }`. Response is
  `text/event-stream` with named events: `token` `{"text":"…"}` (0..n, append in
  order), `done` `{}`, `error` `{"error":"…"}`. `429` (JSON `{"error":"rate_limited"}`)
  before the stream starts when over the per-IP cap.
- `POST {NEXT_PUBLIC_API_BASE}/signal` body `{ "kind":"email", "email" }` or
  `{ "kind":"thumbs", "thumbs":"up"|"down" }` (optional `inputExcerpt`) →
  `201` JSON `{ "status": "recorded" }`. `400` JSON `{ "error":"…" }` on validation.

## The funnel (emitted by construction)

`poc_visit` (landing mount) → `poc_generate` (submit) → `poc_output_action`
(copy/download) → `poc_signal` (email/thumbs). Headline metric = `poc_signal / poc_visit`.

## Config & run

Set `NEXT_PUBLIC_API_BASE` (and optional `ANALYTICS_FORWARD_URL`) — see
`.env.example`. `npm install && npm run dev`. Build is validated in CI.
