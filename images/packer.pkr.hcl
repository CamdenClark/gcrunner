packer {
  required_plugins {
    googlecompute = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/googlecompute"
    }
  }
}

locals {
  timestamp  = formatdate("YYYYMMDDhhmmss", timestamp())
  image_name = "${var.image_family}-${local.timestamp}"
}

source "googlecompute" "runner" {
  project_id          = var.project_id
  zone                = var.zone
  source_image_family = var.base_image_family
  source_image_project_id = [var.base_image_project]
  image_name          = local.image_name
  image_family        = var.image_family
  image_description   = var.image_description
  machine_type        = var.machine_type
  disk_size           = var.disk_size
  ssh_username        = "packer"
  tags                = ["packer"]
}

build {
  sources = ["source.googlecompute.runner"]

  provisioner "shell" {
    script = "scripts/base.sh"
  }

  provisioner "shell" {
    script = "scripts/runner-agent.sh"
    environment_vars = [
      "RUNNER_VERSION=${var.runner_version}",
    ]
  }

  provisioner "shell" {
    script = "scripts/tools.sh"
  }

  provisioner "shell" {
    script = "scripts/gcloud.sh"
  }

  provisioner "shell" {
    script = "scripts/cleanup.sh"
  }
}
