resource "google_secret_manager_secret" "app_id" {
  secret_id = "gcrunner-app-id"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis["secretmanager.googleapis.com"]]
}

resource "google_secret_manager_secret" "private_key" {
  secret_id = "gcrunner-private-key"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis["secretmanager.googleapis.com"]]
}

resource "google_secret_manager_secret" "webhook_secret" {
  secret_id = "gcrunner-webhook-secret"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis["secretmanager.googleapis.com"]]
}

resource "random_password" "setup_token" {
  length  = 32
  special = false
}

resource "google_secret_manager_secret" "setup_token" {
  secret_id = "gcrunner-setup-token"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis["secretmanager.googleapis.com"]]
}

resource "google_secret_manager_secret_version" "setup_token" {
  secret      = google_secret_manager_secret.setup_token.id
  secret_data = random_password.setup_token.result
}
