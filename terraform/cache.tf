resource "google_storage_bucket" "cache" {
  count = var.enable_cache ? 1 : 0

  name                        = var.cache_bucket_name != "" ? var.cache_bucket_name : "${var.project_id}-gcrunner-cache"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = true

  lifecycle_rule {
    condition {
      age = 14
    }
    action {
      type = "Delete"
    }
  }
}
