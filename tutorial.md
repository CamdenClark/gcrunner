# Deploy gcrunner on Google Cloud

<walkthrough-tutorial-duration duration="15"></walkthrough-tutorial-duration>

## Overview

gcrunner is an open-source, drop-in replacement for GitHub-hosted Actions runners on Google Cloud. It provisions ephemeral Compute Engine VMs on demand, runs the job, and self-destructs — giving you full `actions/runner-images` compatibility at **80%+ cost savings**.

**What you'll do:**

- Select a GCP project and enable required APIs
- Deploy infrastructure with Terraform
- Set up a GitHub App to connect your repositories
- Configure your first workflow to use gcrunner

**What you'll need:**

- A GCP project with billing enabled
- A GitHub account with admin access to the repositories you want to use
- Terraform installed (this tutorial will install it if needed)

Click **Start** to begin.

## Project setup

<walkthrough-project-setup billing="true"></walkthrough-project-setup>

First, select a Google Cloud project to deploy gcrunner into. If you don't have one, create a new project using the widget above.

Set the project ID for use throughout this tutorial:

```sh
export PROJECT_ID=<walkthrough-project-id/>
gcloud config set project $PROJECT_ID
```

### Enable required APIs

<walkthrough-enable-apis apis="run.googleapis.com,compute.googleapis.com,secretmanager.googleapis.com,artifactregistry.googleapis.com"></walkthrough-enable-apis>

gcrunner needs Cloud Run, Compute Engine, Secret Manager, and Artifact Registry. Click the button above or run:

```sh
gcloud services enable \
  run.googleapis.com \
  compute.googleapis.com \
  secretmanager.googleapis.com \
  artifactregistry.googleapis.com
```

## Clone the repository

Clone the gcrunner repository and navigate to it:

```sh
git clone https://github.com/camdenclark/gcrunner.git
cd gcrunner
```

## Install Terraform

Check if Terraform is already installed:

```sh
terraform version || true
```

If Terraform is not installed, install it now:

```sh
sudo apt-get update && sudo apt-get install -y gnupg software-properties-common
wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt-get update && sudo apt-get install -y terraform
```

## Deploy infrastructure with Terraform

Navigate to the Terraform directory and initialize:

```sh
cd terraform
terraform init
```

### Configure variables

Set the required Terraform variables:

```sh
export TF_VAR_project_id=$PROJECT_ID
export TF_VAR_region=us-central1
```

To enable the GCS build cache (optional but recommended):

```sh
export TF_VAR_enable_cache=true
```

### Apply the configuration

Review the planned changes and deploy:

```sh
terraform plan
```

If everything looks good, apply:

```sh
terraform apply
```

Type `yes` when prompted to confirm.

Terraform will create:

| Resource | Purpose |
|---|---|
| Cloud Run service | Webhook receiver and VM orchestrator |
| Service accounts | Minimal IAM for function and runner VMs |
| Artifact Registry | Remote repository for the gcrunner container image |
| Secret Manager secrets | GitHub App credentials storage |
| VPC + Firewall | Network isolation for runner VMs |
| GCS bucket | Build cache storage (if enabled) |

### Save the outputs

Once Terraform completes, note the important outputs:

```sh
terraform output
```

The `setup_url` is what you'll use in the next step to create the GitHub App.

```sh
export SETUP_URL=$(terraform output -raw setup_url)
echo "Setup URL: $SETUP_URL"
```

## Set up the GitHub App

gcrunner uses a GitHub App to receive webhook events and register runners. The setup flow is built into gcrunner itself.

### Open the setup page

Open the setup URL in your browser:

```sh
echo $SETUP_URL
```

Click the URL in the terminal output, or copy it into your browser.

### Create and install the GitHub App

1. On the setup page, click **Create GitHub App**
2. GitHub will prompt you to name the app and select where to install it
3. Choose the organization or account where your repositories live
4. Select which repositories should have access to gcrunner
5. Complete the installation

The setup flow will automatically store the GitHub App credentials (app ID, private key, and webhook secret) in Secret Manager.

## Configure your first workflow

Navigate back to the repository root:

```sh
cd ~/gcrunner
```

In any repository where you installed the GitHub App, update your workflow to use gcrunner. Here's an example:

```yaml
name: CI

on: [push, pull_request]

jobs:
  build:
    # This single line is all the configuration you need
    runs-on: gcrunner=${{ github.run_id }}

    steps:
      - uses: actions/checkout@v4
      - run: echo "Running on gcrunner!"
```

### Customizing your runner

All runner configuration lives in the `runs-on` field using simple labels:

```yaml
# Default: n2d-standard-2 spot VM, pd-ssd, 75gb disk
runs-on: gcrunner=${{ github.run_id }}

# Specify machine type
runs-on: gcrunner=${{ github.run_id }}/machine=n2-standard-4

# Bigger machine, more disk
runs-on: gcrunner=${{ github.run_id }}/machine=c3-standard-8/disk=200gb

# No spot for production deploys
runs-on: gcrunner=${{ github.run_id }}/spot=false
```

Available labels:

| Label | Default | Description |
|---|---|---|
| `machine` | `n2d-standard-2` | GCE machine type |
| `spot` | `true` | Use Spot VMs (60-91% cheaper), auto-fallback to on-demand |
| `disk` | `75gb` | Boot disk size |
| `disk-type` | `pd-ssd` | Boot disk type |
| `image` | `ubuntu24-full-x64` | Runner image family |

## Verify the deployment

To verify that everything is working, trigger a workflow in one of your configured repositories. You can check the Cloud Run logs to see webhook events being processed:

```sh
gcloud run services logs read gcrunner-webhook --region=$TF_VAR_region --limit=20
```

You can also list any runner VMs that are currently active:

```sh
gcloud compute instances list --filter="name~gcrunner"
```

## Conclusion

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

You've successfully deployed gcrunner on Google Cloud!

**What you've set up:**

- Cloud Run service receiving GitHub webhook events
- Ephemeral Compute Engine VMs as GitHub Actions runners
- Secure credential storage in Secret Manager
- Optional GCS build caching

**Next steps:**

- Add gcrunner labels to more of your workflows
- Explore different [machine types](https://cloud.google.com/compute/docs/machine-types) for your workloads
- Enable the GCS cache with `extras=gcs-cache` for faster builds
- Check [gcrunner.com](https://gcrunner.com) for full documentation

**Cost savings tip:** Spot VMs (enabled by default) save 60-91% compared to on-demand pricing. For production deploys where you can't risk preemption, use `spot=false`.
