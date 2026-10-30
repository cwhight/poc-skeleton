# Terraform variables — parameterized by var.slug.
#
# var.slug is THE parameter — matches idea.config.slug and namespaces every
# resource (Cloud Run service, Neon DB, Vercel project, analytics, secrets).
# A new POC stands up by setting one variable.

variable "slug" {
  type        = string
  description = "POC slug; namespaces Cloud Run service, Neon DB, Vercel project, secrets. Matches idea.config.slug."

  validation {
    # Cloud Run / DNS / Secret Manager friendly: lower-kebab, starts with a letter.
    condition     = can(regex("^[a-z][a-z0-9-]{1,30}$", var.slug))
    error_message = "slug must be lower-case alphanumeric/hyphen, start with a letter, 2-31 chars."
  }
}

# ---- GCP (Cloud Run + Secret Manager) -------------------------------------

variable "gcp_project_id" {
  type        = string
  description = "GCP project id that hosts Cloud Run + Secret Manager. PLACEHOLDER — set per POC. (GCP work uses the project's designated account.)"
}

variable "region" {
  type        = string
  description = "Region for Cloud Run and (ideally) Neon. Match the Neon region to minimise latency."
  default     = "europe-west1"
}

variable "api_image" {
  type        = string
  description = "Fully-qualified container image for the Go API, e.g. europe-west1-docker.pkg.dev/PROJECT/poc/SLUG-api:TAG. CI builds+pushes this — agents never build images locally."
  # No default: CI passes the freshly-pushed image digest/tag via -var.
}

# ---- LLM provider (vendor-neutral, swappable) -----------------------------

variable "llm_provider" {
  type        = string
  description = "Selects the provider impl in api/internal/llm (e.g. gemini, openai, anthropic). Only this provider is billed."
  default     = "gemini"
}

variable "llm_model" {
  type        = string
  description = "Model id for the selected provider. Vendor-specific; user-facing copy stays vendor-neutral."
}

# ---- Tunables (cost / abuse caps) -----------------------------------------

variable "max_input_bytes" {
  type        = number
  description = "Server-side input size cap enforced in the Go handler (protects cost + latency)."
  default     = 12000
}

variable "rate_limit_per_ip" {
  type        = number
  description = "Max paid (/generate) calls per IP per window. An uncapped AI endpoint is an open invoice."
  default     = 20
}

variable "rate_limit_window" {
  type        = string
  description = "Rate-limit window as a Go duration string, e.g. 1h, 24h."
  default     = "1h"
}

# ---- Secrets (values live in Secret Manager / Neon / Vercel, NOT here) -----

variable "llm_api_key" {
  type        = string
  sensitive   = true
  description = "API key for the selected LLM provider. Supplied via TF_VAR_llm_api_key (CI secret) — never committed; stored in Secret Manager."
}

variable "neon_api_key" {
  type        = string
  sensitive   = true
  description = "Neon API key used by the Neon provider to manage the project/branch/role. Supplied via TF_VAR_neon_api_key (CI secret)."
}

variable "neon_org_id" {
  type        = string
  description = "Neon organization id that owns the project. PLACEHOLDER — set per POC."
}

variable "vercel_api_token" {
  type        = string
  sensitive   = true
  description = "Vercel API token used by the Vercel provider. Supplied via TF_VAR_vercel_api_token (CI secret)."
}

variable "vercel_team_id" {
  type        = string
  description = "Vercel team/org id that owns the project. PLACEHOLDER — set per POC (or null for personal scope)."
  default     = null
}

variable "web_git_repo" {
  type        = string
  description = "GitHub repo for the Next.js web in owner/name form, used to wire Vercel git deploys. PLACEHOLDER — set per POC."
}
