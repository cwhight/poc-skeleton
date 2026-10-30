# Patterns Baked In

The skeleton enforces these LEARNINGS **by construction** so every POC — and every build agent that fills in a stub — inherits them without re-deriving them. Each is a hard requirement, not a suggestion. Distilled from `templates/new-company/LEARNINGS.md`.

## Correctness
- [ ] **HTML-escape before rendering, then layer only your own tags.** The `OutputView` component escapes model output *before* markdown rendering — never `dangerouslySetInnerHTML` raw LLM text. Applies on *every* surface that shows generated content.
- [ ] **Markdown-safe render is consistent across surfaces.** One render path (`OutputView`) for the generated artifact; no surface shows raw `##`/`**`.
- [ ] **Server-side validation is authoritative.** Input non-empty + within `MAX_INPUT_BYTES` is enforced in the Go handler, not just the browser. The client can lie.
- [ ] **Input size limits.** `maxInputBytes` from the config caps the request server-side (protects cost and latency).

## Distributed systems & cost
- [ ] **DB-backed rate limit (shared store).** Per-IP counters live in Postgres, not process memory — the cap holds across Cloud Run instances and survives deploys. An in-memory limiter on autoscaling = N× the cap and resets every deploy.
- [ ] **Rate-limit the paid endpoint, count only paid calls.** `/generate` (the LLM call that costs money) is capped per IP per window. An uncapped AI endpoint is an open invoice.
- [ ] **Trust the correct XFF hop.** Client IP is resolved from the trustworthy platform-appended hop, not the spoofable leftmost `X-Forwarded-For` entry.
- [ ] **No write deadline on the SSE stream.** The streaming response disables (or sets a long) write timeout so a default server deadline can't kill the stream mid-flight after headers are sent.

## Auth & security
- [ ] **Anonymous-first; no auth in the core flow.** Validation needs zero friction — the paste→generate→signal path requires no login.
- [ ] **If any protected/admin route is added, it fails CLOSED.** A thrown session check redirects to sign-in, never falls through to render.

## LLM
- [ ] **Vendor-neutral, swappable provider behind an interface.** `llm.Provider` interface; concrete impl chosen by `LLM_PROVIDER` env var. No provider SDK hard-wired across the codebase.
- [ ] **User-facing copy is vendor-neutral.** Landing/output copy never names the model — the model you launch on is rarely the one you keep.

## Product & launch (honest by construction)
- [ ] **Honest, verifiable copy.** No invented metrics, testimonials, or user counts in landing copy. No claim the flow can't deliver.
- [ ] **Hypothesis + success metric + kill condition, decided before launch.** Required fields in the idea-config (`hypothesis`, `successMetric`, `killCondition`).
- [ ] **The funnel is the experiment.** The four events (`poc_visit`, `poc_generate`, `poc_output_action`, `poc_signal`) are emitted by construction — see README's funnel table.

## Design
- [ ] **Icon library, not emoji/unicode glyphs** (lucide) for UI affordances.
- [ ] **Constrained palette** — a small set of semantic colours + grays.
- [ ] **≥44px tap targets**, including secondary controls (back-links, badges, icon buttons), not just the primary CTA.
- [ ] **Mobile-first viewport check** — every screen tested at a phone width for overflow/truncation.

## Process & infra
- [ ] **Infrastructure as code only.** All infra + env vars in Terraform (`infra/`), parameterized by `slug`. No click-ops. Deploy from `main` via CI.
- [ ] **branch → PR → CI-green → different-agent review → squash-merge → verify live.** The author never reviews/merges its own work.
- [ ] **CI test-gate is a hard merge gate.** Tests must pass before merge; `.github/workflows` enforces it.

---

*Build agents: when you fill a stub, tick the boxes your file touches. Reviewers: this is the checklist the review verifies against.*
