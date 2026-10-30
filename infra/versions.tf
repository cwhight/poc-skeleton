# Provider + backend pinning.
#
# REMOTE STATE: GCS backend so state is shared and locked. `terraform apply` is
# run by a human or by CI from `main` ONLY (IaC-only, no click-ops). Never apply
# from a feature branch or a laptop pointed at prod.
#
# Backend config is partial on purpose — the bucket/prefix are passed at init
# time so the same template serves every POC:
#   terraform init \
#     -backend-config="bucket=YOUR_TF_STATE_BUCKET" \
#     -backend-config="prefix=poc/${var.slug}"
terraform {
  required_version = ">= 1.6.0"

  backend "gcs" {
    # bucket + prefix supplied via -backend-config at init (see above).
    # PLACEHOLDER defaults left blank so a misconfigured init fails loudly.
  }

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    neon = {
      source  = "kislerdm/neon"
      version = "~> 0.6"
    }
    vercel = {
      source  = "vercel/vercel"
      version = "~> 2.0"
    }
  }
}

provider "google" {
  project = var.gcp_project_id
  region  = var.region
}

provider "neon" {
  api_key = var.neon_api_key
}

provider "vercel" {
  api_token = var.vercel_api_token
  team      = var.vercel_team_id
}
