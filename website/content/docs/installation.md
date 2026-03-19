---
title: "Installation"
weight: 2
---

# Installation

## Cloud Shell (recommended)

The fastest way to deploy gcrunner is with Google Cloud Shell. It provides a pre-configured environment with everything you need.

<a href="https://shell.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https%3A%2F%2Fgithub.com%2Fcamdenclark%2Fgcrunner&cloudshell_tutorial=tutorial.md" target="_blank" rel="noopener noreferrer">
  <img src="https://gstatic.com/cloudssh/images/open-btn.svg" alt="Open in Cloud Shell">
</a>

This opens an interactive tutorial that walks you through the full setup in about 15 minutes.

### What you'll need

- A **new GCP project** with billing enabled
- A GitHub account with a repository to test against

### What gets created

| Resource | Purpose |
|---|---|
| Cloud Run service | Webhook receiver and VM orchestrator |
| Service accounts | Minimal IAM for the service and runner VMs |
| Artifact Registry | Remote repository for the gcrunner container image |
| Secret Manager secrets | GitHub App credentials storage |
| GCS bucket | Build cache storage |

## Manual setup

If you prefer to deploy manually, you can use Terraform directly.

### Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) installed
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) installed and authenticated
- A GCP project with billing enabled

### Deploy

```sh
git clone https://github.com/camdenclark/gcrunner.git
cd gcrunner/terraform
terraform init
terraform apply -var="project_id=YOUR_PROJECT_ID"
```

### Set up the GitHub App

After Terraform completes, get the setup URL:

```sh
terraform output -raw setup_url
```

Open the URL in your browser, then:

1. Click **Create GitHub App**
2. Name the app and select where to install it
3. Select which repositories should have access to gcrunner
4. Complete the installation

The credentials (app ID, private key, webhook secret) are automatically stored in Secret Manager.

## Configure your first workflow

In any repository where you installed the GitHub App, use gcrunner in your workflow:

```yaml
name: CI

on: [push, pull_request]

jobs:
  build:
    runs-on: gcrunner=${{ github.run_id }}
    steps:
      - uses: actions/checkout@v4
      - run: echo "Running on gcrunner!"
```

### Customizing your runner

All configuration lives in the `runs-on` field using `/`-separated labels:

```yaml
# Bigger machine, more disk
runs-on: gcrunner=${{ github.run_id }}/machine=c3-standard-8/disk=200gb

# No spot for production deploys
runs-on: gcrunner=${{ github.run_id }}/spot=false
```

| Label | Default | Description |
|---|---|---|
| `machine` | `n2d-standard-2` | GCE machine type |
| `spot` | `true` | Use Spot VMs (60-91% cheaper), auto-fallback to on-demand |
| `disk` | `75gb` | Boot disk size |
| `disk-type` | `pd-ssd` | Boot disk type |
| `image` | `ubuntu24-full-x64` | Runner image family |
