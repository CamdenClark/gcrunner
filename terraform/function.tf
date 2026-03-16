data "archive_file" "function_source" {
  type        = "zip"
  source_dir  = "${path.module}/../function"
  output_path = "${path.module}/.build/function-source.zip"
}

resource "google_storage_bucket" "function_source" {
  name                        = "${var.project_id}-gcrunner-source"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = true
}

resource "google_storage_bucket_object" "function_source" {
  name   = "function-source-${data.archive_file.function_source.output_md5}.zip"
  bucket = google_storage_bucket.function_source.name
  source = data.archive_file.function_source.output_path
}

resource "google_cloudfunctions2_function" "webhook" {
  name     = var.function_name
  location = var.region

  build_config {
    runtime     = "go126"
    entry_point = "Webhook"

    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = google_storage_bucket_object.function_source.name
      }
    }
  }

  service_config {
    service_account_email = google_service_account.function.email
    max_instance_count    = var.max_instances
    available_memory      = "256Mi"
    timeout_seconds       = 120

    environment_variables = {
      GCP_PROJECT           = var.project_id
      GCE_REGION            = var.region
      GCRUNNER_CACHE_BUCKET = var.enable_cache ? (var.cache_bucket_name != "" ? var.cache_bucket_name : "${var.project_id}-gcrunner-cache") : ""
      GCRUNNER_IMAGE_PROJECT = var.image_project
    }
  }

  depends_on = [
    google_project_service.apis["cloudfunctions.googleapis.com"],
    google_project_service.apis["cloudbuild.googleapis.com"],
    google_project_service.apis["run.googleapis.com"],
    google_project_service.apis["artifactregistry.googleapis.com"],
  ]
}

# Allow unauthenticated access (GitHub webhooks)
resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  project  = var.project_id
  location = var.region
  name     = google_cloudfunctions2_function.webhook.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
