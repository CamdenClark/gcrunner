# gcrunner

Self-hosted GitHub Actions runners on Google Cloud. Drop-in replacement for GitHub-hosted runners at 80%+ cost savings.

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://shell.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https%3A%2F%2Fgithub.com%2Fcamdenclark%2Fgcrunner&cloudshell_git_branch=v0.2.0&cloudshell_tutorial=tutorial.md)

## How it works

gcrunner receives GitHub Actions `workflow_job` webhooks via a Cloud Run service, provisions an ephemeral Compute Engine VM with your requested specs, runs the job, and self-destructs the VM when done.

```
GitHub Actions → Webhook → Cloud Run (Go) → Compute Engine VM → Self-destruct
```

## Quick start

All configuration lives in your workflow's `runs-on` field:

```yaml
jobs:
  build:
    runs-on: gcrunner=${{ github.run_id }}
    steps:
      - uses: actions/checkout@v4
      - run: echo "Running on gcrunner!"
```

### Labels

| Label | Default | Description |
|---|---|---|
| `machine` | `n2d-standard-2` | GCE machine type |
| `spot` | `true` | Spot VMs (60-91% cheaper), auto-fallback to on-demand |
| `disk` | `75gb` | Boot disk size |
| `disk-type` | `pd-ssd` | Boot disk type |
| `image` | `ubuntu24-full-x64` | Runner image family |

```yaml
# Bigger machine, more disk
runs-on: gcrunner=${{ github.run_id }}/machine=c3-standard-8/disk=200gb

# No spot for production deploys
runs-on: gcrunner=${{ github.run_id }}/spot=false
```

## Deploying

Clone the latest release and deploy with Terraform:

```sh
git clone --branch v0.2.0 https://github.com/camdenclark/gcrunner
cd gcrunner/terraform
gcloud storage buckets create gs://YOUR_PROJECT_ID-gcrunner-tfstate --location=us-central1
terraform init -backend-config="bucket=YOUR_PROJECT_ID-gcrunner-tfstate" -backend-config="prefix=gcrunner"
terraform apply -var="project_id=YOUR_PROJECT_ID"
```

Then visit the setup URL from the Terraform output to create and install the GitHub App.

For a guided walkthrough, use the **Open in Cloud Shell** button above.

## Documentation

Visit [gcrunner.com](https://gcrunner.com) for full documentation.
