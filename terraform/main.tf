terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

locals {
  apis = toset([
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "compute.googleapis.com",
  ])
}

resource "google_project_service" "apis" {
  for_each           = local.apis
  service            = each.value
  disable_on_destroy = false
}
