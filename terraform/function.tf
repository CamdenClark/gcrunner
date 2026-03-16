resource "google_artifact_registry_repository" "ghcr_remote" {
  repository_id = "gcrunner-ghcr"
  location      = var.region
  format        = "DOCKER"
  mode          = "REMOTE_REPOSITORY"

  remote_repository_config {
    docker_repository {
      custom_repository {
        uri = "https://ghcr.io"
      }
    }
  }

  depends_on = [
    google_project_service.apis["artifactregistry.googleapis.com"],
  ]
}

resource "google_cloud_run_v2_service" "webhook" {
  name     = var.function_name
  location = var.region

  template {
    service_account = google_service_account.function.email

    containers {
      image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.ghcr_remote.repository_id}/camdenclark/gcrunner:latest"

      resources {
        limits = {
          memory = "512Mi"
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
    google_artifact_registry_repository.ghcr_remote,
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
