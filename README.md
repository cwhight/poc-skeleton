# POC Skeleton — turn an idea into a validation POC in hours

A **parameterized product template**. Every POC we build has the same shape; the only thing that changes per idea is a single config file and a system prompt. Fill those in, deploy, and watch the funnel.

## The shape of every POC

```
landing page  →  user pastes input  →  LLM generates (streamed)  →  primary action (copy/download)
                                                                  →  validation signal (email capture / thumbs)
```

A POC exists to **validate the riskiest "would the user trust and use this?" assumption — cheaply**. So:

- **Anonymous-first.** No signup wall in front of the core flow. Friction kills validation signal.
- **One job.** Paste in → get a useful artifact out. Nothing else.
- **Capture one signal.** After the user sees value, ask for the validation signal: an email ("send me this / notify me at launch") or a thumbs + a one-line "would you use this?" question. That, plus the analytics funnel, is the experiment result.

## New idea in 4 steps

1. **Fill the idea-config.** Copy `idea.config.example.ts` to `idea.config.ts` and fill every field (name, slug, tagline, landing copy, input label/placeholder, the generate `systemPrompt`, output format, CTA text, signal question). This file *is* the product definition — see the two worked examples (`configs/scopelock.config.ts`, `configs/reviewpilot.config.ts`).
2. **Set the prompt.** The `systemPrompt` field is the generate prompt. Iterate it against real inputs until the output is something a user would actually copy/download. This is the highest-leverage tuning — spend time here.
3. **Deploy.** `slug` parameterizes everything: Vercel project, Cloud Run service, Neon DB name, analytics namespace. Run the Terraform in `infra/` (IaC only — never click-ops), push to `main`, CI deploys.
4. **Watch the funnel.** Open the analytics dashboard and read the validation-funnel metric below. A POC has a **decision deadline** and a **kill condition** — decide both before launch.

## The validation funnel (the metric that matters)

Every POC reports the same funnel. These four events are emitted by construction:

| Step | Event | Question it answers |
|------|-------|---------------------|
| 1. Visit | `poc_visit` | Did the positioning get them to the page? |
| 2. Generate | `poc_generate` | Was the promise compelling enough to paste input and hit generate? |
| 3. Copy / download | `poc_output_action` | Was the output good enough to *take away* and use? |
| 4. "I'd use this" | `poc_signal` (email captured / thumbs-up) | Would they actually adopt it? |

**The headline conversion is `poc_signal / poc_visit`** (and the step-to-step drop-offs tell you *where* the idea fails). Define the success threshold and the kill condition in the idea-config's accompanying brief **before** launch — `poc_output_action → poc_signal` is the truth serum: people who took the output but won't leave an email don't actually want it.

## What's in here

| Path | What it is |
|------|------------|
| `ARCHITECTURE.md` | Stack, folder layout, request flow. |
| `idea.config.example.ts` | The single idea-config schema — the one artifact a new POC fills in. |
| `configs/scopelock.config.ts` | Worked example: freelancer contract from a brief. |
| `configs/reviewpilot.config.ts` | Worked example: manager performance-review from notes. |
| `PATTERNS-BAKED-IN.md` | The LEARNINGS this skeleton enforces by construction — build agents inherit them. |
| `web/` | Next.js app (landing + paste→generate page). Stubs; build agent fills. |
| `api/` | Go API (`/generate` SSE, DB-backed rate limit, swappable LLM interface). Stubs. |
| `infra/` | Terraform, parameterized by `slug`. Stubs. |
| `.github/` | CI (test-gate + deploy). Stubs. |

> Files in `web/`, `api/`, `infra/`, `.github/` are **stubs with header comments**. A build agent fills each one (routing boilerplate via the cheap model). This README + `ARCHITECTURE.md` + the idea-config are the spec they build against.
