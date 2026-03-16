variable "image" {
  description = "Container image for the webhook service"
  type        = string
  default     = "ghcr.io/camdenclark/gcrunner:latest"
}

resource "google_cloud_run_v2_service" "webhook" {
  name     = var.function_name
  location = var.region

  template {
    service_account = google_service_account.function.email

    scaling {
      max_instance_count = var.max_instances
    }

    containers {
      image = var.image

      resources {
        limits = {
          memory = "256Mi"
          cpu    = "1"
        }
      }

      env {
        name  = "GCP_PROJECT"
        value = var.project_id
      }
      env {
        name  = "GCE_REGION"
        value = var.region
      }
      env {
        name  = "GCRUNNER_CACHE_BUCKET"
        value = var.enable_cache ? (var.cache_bucket_name != "" ? var.cache_bucket_name : "${var.project_id}-gcrunner-cache") : ""
      }
      env {
        name  = "GCRUNNER_IMAGE_PROJECT"
        value = var.image_project
      }
    }
  }

  depends_on = [
    google_project_service.apis["run.googleapis.com"],
  ]
}

# Allow unauthenticated access (GitHub webhooks)
resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.webhook.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
