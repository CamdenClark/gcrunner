output "webhook_url" {
  value = google_cloud_run_v2_service.webhook.uri
}

output "setup_url" {
  value = "${google_cloud_run_v2_service.webhook.uri}/setup"
}

output "function_service_account" {
  value = google_service_account.function.email
}

output "runner_service_account" {
  value = google_service_account.runner.email
}

output "cache_bucket_name" {
  value = var.enable_cache ? google_storage_bucket.cache[0].name : ""
}
