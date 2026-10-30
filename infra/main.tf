# Infrastructure for one POC, fully namespaced by var.slug.
# A new POC = set var.slug (+ secrets) and apply. No click-ops; CI applies from main.
#
# Provisions: Neon Postgres (pooler endpoint), Secret Manager (DATABASE_URL +
# LLM_API_KEY), a Cloud Run service for the Go API, its IAM, and a Vercel project
# for the Next.js web. All env vars are set HERE — nothing set outside Terraform.

locals {
  name = var.slug # single source of truth for every resource name

  # Plain (non-secret) env vars handed to the Cloud Run container.
  api_env = {
    POC_SLUG          = var.slug
    LLM_PROVIDER      = var.llm_provider
    LLM_MODEL         = var.llm_model
    MAX_INPUT_BYTES   = tostring(var.max_input_bytes)
    RATE_LIMIT_PER_IP = tostring(var.rate_limit_per_ip)
    RATE_LIMIT_WINDOW = var.rate_limit_window
  }
}

# ---------------------------------------------------------------------------
# Neon Postgres — serverless; free tier covers a POC.
# LEARNING: DATABASE_URL must use the *pooler* endpoint (connection pooling),
# not the direct host. Cloud Run scales to many short-lived instances; the
# pooler keeps Postgres connection count sane. The pooled host is the
# direct host with "-pooler" inserted before the first dot.
# ---------------------------------------------------------------------------
resource "neon_project" "this" {
  name      = local.name
  org_id    = var.neon_org_id
  region_id = "aws-${var.region}" # Neon region ids are aws-prefixed; align with GCP region where possible.

  # Free-tier friendly autoscaling.
  branch {
    name          = "main"
    database_name = replace(local.name, "-", "_") # Postgres db names can't contain hyphens.
    role_name     = "${replace(local.name, "-", "_")}_app"
  }
}

locals {
  neon_user = neon_project.this.database_user
  neon_pass = neon_project.this.database_password
  neon_db   = neon_project.this.database_name

  # neon_project exposes the direct host; derive the pooled host for DATABASE_URL.
  # The pooled host is the direct host with "-pooler" appended to the first label
  # (the endpoint id), e.g. ep-xxx.eu... -> ep-xxx-pooler.eu...
  neon_direct_host = neon_project.this.database_host
  neon_pooler_host = replace(local.neon_direct_host, "/^([^.]+)/", "$1-pooler")

  database_url = "postgresql://${local.neon_user}:${local.neon_pass}@${local.neon_pooler_host}/${local.neon_db}?sslmode=require"
}

# ---------------------------------------------------------------------------
# Secret Manager — DATABASE_URL + LLM_API_KEY. Secrets never sit in env literals
# in the Cloud Run config; they are mounted from Secret Manager versions.
# ---------------------------------------------------------------------------
resource "google_secret_manager_secret" "database_url" {
  secret_id = "${local.name}-database-url"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "database_url" {
  secret      = google_secret_manager_secret.database_url.id
  secret_data = local.database_url
}

resource "google_secret_manager_secret" "llm_api_key" {
  secret_id = "${local.name}-llm-api-key"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "llm_api_key" {
  secret      = google_secret_manager_secret.llm_api_key.id
  secret_data = var.llm_api_key
}

# ---------------------------------------------------------------------------
# Service account for the API + least-privilege IAM.
# ---------------------------------------------------------------------------
resource "google_service_account" "api" {
  account_id   = "${local.name}-api"
  display_name = "${local.name} POC API (Cloud Run)"
}

# The runtime SA may read ONLY this POC's two secrets.
resource "google_secret_manager_secret_iam_member" "database_url_accessor" {
  secret_id = google_secret_manager_secret.database_url.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api.email}"
}

resource "google_secret_manager_secret_iam_member" "llm_api_key_accessor" {
  secret_id = google_secret_manager_secret.llm_api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api.email}"
}

# ---------------------------------------------------------------------------
# Cloud Run — the Go API. Scales to zero; streaming-friendly.
# ---------------------------------------------------------------------------
resource "google_cloud_run_v2_service" "api" {
  name     = "${local.name}-api"
  location = var.region

  template {
    service_account = google_service_account.api.email

    # Scale to zero; cap to protect cost. Rate limiting is DB-backed so the cap
    # holds across however many instances autoscaling spins up.
    scaling {
      min_instance_count = 0
      max_instance_count = 4
    }

    containers {
      image = var.api_image

      ports {
        container_port = 8080 # api/cmd/main.go listens on $PORT (Cloud Run sets PORT=8080)
      }

      # Non-secret env.
      dynamic "env" {
        for_each = local.api_env
        content {
          name  = env.key
          value = env.value
        }
      }

      # Secret env — sourced from Secret Manager, not literals.
      env {
        name = "DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.database_url.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "LLM_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.llm_api_key.secret_id
            version = "latest"
          }
        }
      }

      startup_probe {
        http_get {
          path = "/healthz"
        }
        initial_delay_seconds = 2
        period_seconds        = 5
        failure_threshold     = 6
      }
    }

    # Streaming (SSE) friendly: a generous request timeout so long token streams
    # are not cut off. (The Go handler sets no write deadline on the stream.)
    timeout = "300s"
  }

  depends_on = [
    google_secret_manager_secret_version.database_url,
    google_secret_manager_secret_version.llm_api_key,
    google_secret_manager_secret_iam_member.database_url_accessor,
    google_secret_manager_secret_iam_member.llm_api_key_accessor,
  ]
}

# Anonymous-first: the core POC flow needs zero auth, so the service is public.
# (Any admin/protected route lives behind app-level checks that fail CLOSED.)
resource "google_cloud_run_v2_service_iam_member" "public" {
  name     = google_cloud_run_v2_service.api.name
  location = google_cloud_run_v2_service.api.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# ---------------------------------------------------------------------------
# Vercel — the Next.js web. Points the browser at the Cloud Run API URL.
# ---------------------------------------------------------------------------
resource "vercel_project" "web" {
  name           = local.name
  framework      = "nextjs"
  root_directory = "web"

  git_repository = {
    type = "github"
    repo = var.web_git_repo
  }
}

# The web reads the API base URL at build time. Vendor-neutral, no secrets.
resource "vercel_project_environment_variable" "api_base_url" {
  project_id = vercel_project.web.id
  key        = "NEXT_PUBLIC_API_BASE_URL"
  value      = google_cloud_run_v2_service.api.uri
  target     = ["production", "preview"]
}

resource "vercel_project_environment_variable" "poc_slug" {
  project_id = vercel_project.web.id
  key        = "NEXT_PUBLIC_POC_SLUG"
  value      = var.slug
  target     = ["production", "preview"]
}

# ---------------------------------------------------------------------------
# Outputs — consumed by deploy.yml for the post-deploy smoke check.
# ---------------------------------------------------------------------------
output "api_url" {
  description = "Public Cloud Run URL of the API; smoke-checked at /healthz after deploy."
  value       = google_cloud_run_v2_service.api.uri
}

output "api_service_name" {
  description = "Cloud Run service name (for gcloud run deploy from CI)."
  value       = google_cloud_run_v2_service.api.name
}

output "web_project_name" {
  description = "Vercel project name for the web."
  value       = vercel_project.web.name
}
