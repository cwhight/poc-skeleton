# infra/ — Terraform

All POC infrastructure as code, parameterized by `var.slug`. A new POC stands up
by setting one variable (plus secrets). **No click-ops.** `terraform apply` runs
by a human or by CI from `main` only — never from a feature branch or a laptop
pointed at prod (see `.github/workflows/deploy.yml`).

## What it provisions (all namespaced by `var.slug`)

| Resource | Name | Notes |
|----------|------|-------|
| Neon Postgres | `<slug>` | Serverless; free tier covers a POC. `DATABASE_URL` uses the **pooler** endpoint — Cloud Run scales to many short-lived instances, so the pooled host keeps Postgres connection count sane. |
| Secret Manager | `<slug>-database-url`, `<slug>-llm-api-key` | Secrets mounted into Cloud Run from Secret Manager — never env literals, never committed. |
| Cloud Run | `<slug>-api` | The Go API. Scales to zero, `max=4`. 300s request timeout for SSE streams. Public (anonymous-first core flow). |
| Service account | `<slug>-api` | Least privilege: read access to this POC's two secrets only. |
| Vercel project | `<slug>` | The Next.js web. Gets `NEXT_PUBLIC_API_BASE_URL` (the Cloud Run URL) + `NEXT_PUBLIC_POC_SLUG`. |

## Files

- `versions.tf` — provider pins + **GCS remote state backend** (partial config; bucket/prefix passed at `init`).
- `variables.tf` — `slug` (validated) + region, llm provider/model, cost caps, and secret/id vars.
- `main.tf` — the resources above, secret wiring, IAM, and outputs (`api_url`, `api_service_name`, `web_project_name`).
- `terraform.tfvars.example` — copy to `terraform.tfvars`; secrets stay out (passed via `TF_VAR_*`).

## Secrets & ids a new POC must set

Non-secret (in `terraform.tfvars`): `slug`, `gcp_project_id`, `region`,
`llm_provider`, `llm_model`, `neon_org_id`, `vercel_team_id`, `web_git_repo`,
and the cost caps. Secrets as env vars (CI secrets — never committed):

```bash
export TF_VAR_llm_api_key=...      # billed LLM provider key  → Secret Manager
export TF_VAR_neon_api_key=...     # Neon API key (manages the project)
export TF_VAR_vercel_api_token=... # Vercel API token
```

## Apply (human or CI, from main only)

```bash
terraform init \
  -backend-config="bucket=YOUR_TF_STATE_BUCKET" \
  -backend-config="prefix=poc/<slug>"
terraform apply -var "api_image=<freshly-pushed image>"   # CI passes the image; agents never build images locally
```

`terraform apply` runs after **every** infra change — never leave drift.
