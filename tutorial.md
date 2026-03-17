# Deploy gcrunner on Google Cloud

<walkthrough-tutorial-duration duration="15"></walkthrough-tutorial-duration>

By the end of this tutorial, you'll have GitHub Actions self-hosted runners running on ephemeral Google Cloud VMs. Every job gets its own fresh VM that boots in ~25 seconds, runs your workflow, and self-destructs — giving you full `ubuntu-latest` compatibility at a fraction of the cost of GitHub-hosted runners.

## Prerequisites

- A **brand new GCP project** with billing enabled — if you just created it, reload this tutorial to see it in the project picker below
- A GitHub account with a repository to test against

<walkthrough-project-setup billing="true"></walkthrough-project-setup>

Click **Next** to begin.

## Project setup

Select your new project above, then set the project ID:

```sh
export PROJECT_ID=<walkthrough-project-id/>
gcloud config set project $PROJECT_ID
```

### Enable required APIs

<walkthrough-enable-apis apis="cloudresourcemanager.googleapis.com"></walkthrough-enable-apis>

## Install Terraform

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

### Apply the configuration

Review and deploy:

```sh
terraform plan && terraform apply
```

Type `yes` when prompted. If you see an error that an API has not been enabled or is disabled, wait a minute and re-run — API enablement can take a moment to propagate.

Terraform will create:

| Resource | Purpose |
|---|---|
| Cloud Run service | Webhook receiver and VM orchestrator |
| Service accounts | Minimal IAM for function and runner VMs |
| Artifact Registry | Remote repository for the gcrunner container image |
| Secret Manager secrets | GitHub App credentials storage |
| GCS bucket | Build cache storage |

## Set up the GitHub App

Get your setup URL and open it in your browser:

```sh
terraform output -raw setup_url
```

1. Click **Create GitHub App**
2. Name the app and select where to install it
3. Select which repositories should have access to gcrunner
4. Complete the installation

The credentials (app ID, private key, webhook secret) are automatically stored in Secret Manager.

## Configure your first workflow

In any repository where you installed the GitHub App, update your workflow to use gcrunner:

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

Trigger a workflow in one of your configured repositories, then check the logs:

```sh
gcloud run services logs read gcrunner-webhook --region=$TF_VAR_region --limit=20
```

List active runner VMs:

```sh
gcloud compute instances list --filter="name~gcrunner"
```

## Conclusion

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

You've deployed gcrunner. Check [gcrunner.com](https://gcrunner.com) for full documentation.
