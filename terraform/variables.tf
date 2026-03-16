variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "enable_cache" {
  description = "Create GCS cache bucket"
  type        = bool
  default     = false
}

variable "cache_bucket_name" {
  description = "Custom cache bucket name (auto-generated if empty)"
  type        = string
  default     = ""
}

variable "image_project" {
  description = "Project with runner VM images"
  type        = string
  default     = "gcrunner-images"
}

variable "function_name" {
  description = "Cloud Function name"
  type        = string
  default     = "gcrunner-webhook"
}

variable "max_instances" {
  description = "Max concurrent function instances"
  type        = number
  default     = 10
}
