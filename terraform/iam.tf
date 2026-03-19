resource "google_service_account" "function" {
  account_id   = "gcrunner-function"
  display_name = "gcrunner Cloud Function"
}

# Hardcoded name — must match orchestrator/vm.go
resource "google_service_account" "runner" {
  account_id   = "gcrunner-runner"
  display_name = "gcrunner VM Runner"
}

# Function SA: create/delete VMs
resource "google_project_iam_member" "function_compute" {
  project = var.project_id
  role    = "roles/compute.instanceAdmin.v1"
  member  = "serviceAccount:${google_service_account.function.email}"
}

# Function SA: read secrets at runtime
resource "google_project_iam_member" "function_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.function.email}"
}

# Function SA: create secret versions during /setup flow
resource "google_project_iam_member" "function_secret_admin" {
  project = var.project_id
  role    = "roles/secretmanager.admin"
  member  = "serviceAccount:${google_service_account.function.email}"
}

# Function SA can attach runner SA to VMs
resource "google_service_account_iam_member" "function_uses_runner" {
  service_account_id = google_service_account.runner.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.function.email}"
}

# Dedicated SA for Cloud Tasks to invoke Cloud Run
resource "google_service_account" "tasks" {
  account_id   = "gcrunner-tasks"
  display_name = "gcrunner Cloud Tasks Invoker"
}

# Tasks SA: invoke Cloud Run
resource "google_cloud_run_v2_service_iam_member" "tasks_invoker" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.webhook.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.tasks.email}"
}

# Function SA: enqueue tasks
resource "google_project_iam_member" "function_tasks_enqueuer" {
  project = var.project_id
  role    = "roles/cloudtasks.enqueuer"
  member  = "serviceAccount:${google_service_account.function.email}"
}

# Function SA can create tasks that impersonate the tasks SA
resource "google_service_account_iam_member" "function_uses_tasks" {
  service_account_id = google_service_account.tasks.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.function.email}"
}

# Runner SA: self-delete VMs
resource "google_project_iam_member" "runner_compute" {
  project = var.project_id
  role    = "roles/compute.instanceAdmin.v1"
  member  = "serviceAccount:${google_service_account.runner.email}"
}

# Runner SA: cache bucket access (conditional)
resource "google_storage_bucket_iam_member" "runner_cache" {
  count  = var.enable_cache ? 1 : 0
  bucket = google_storage_bucket.cache[0].name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.runner.email}"
}
