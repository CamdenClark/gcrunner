variable "project_id" {
  type        = string
  description = "GCP project ID to build the image in"
}

variable "zone" {
  type        = string
  default     = "us-central1-a"
  description = "GCE zone to use for the build VM"
}

variable "base_image_family" {
  type        = string
  description = "Base image family (e.g. ubuntu-2404-lts-amd64)"
}

variable "base_image_project" {
  type        = string
  default     = "ubuntu-os-cloud"
  description = "Project containing the base image"
}

variable "image_family" {
  type        = string
  description = "Image family name for the output image"
}

variable "image_description" {
  type        = string
  default     = "gcrunner pre-built runner image"
  description = "Description for the output image"
}

variable "machine_type" {
  type        = string
  default     = "e2-standard-4"
  description = "Machine type for the build VM"
}

variable "disk_size" {
  type        = number
  default     = 50
  description = "Boot disk size in GB"
}

variable "runner_version" {
  type        = string
  default     = ""
  description = "GitHub Actions runner version (empty = latest)"
}
