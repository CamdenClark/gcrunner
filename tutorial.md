# Deploy gcrunner on Google Cloud

<walkthrough-tutorial-duration duration="15"></walkthrough-tutorial-duration>

Click **Start** to begin.

## Project setup

<walkthrough-project-setup billing="true"></walkthrough-project-setup>

Select a Google Cloud project above, then set the project ID:

```sh
export PROJECT_ID=<walkthrough-project-id/>
gcloud config set project $PROJECT_ID
```

### Enable required APIs

<walkthrough-enable-apis apis="run.googleapis.com,compute.googleapis.com,secretmanager.googleapis.com,artifactregistry.googleapis.com,cloudresourcemanager.googleapis.com"></walkthrough-enable-apis>

```sh
gcloud services enable \
  run.googleapis.com \
  compute.googleapis.com \
  secretmanager.googleapis.com \
  artifactregistry.googleapis.com \
  cloudresourcemanager.googleapis.com
```

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
cd ~/cloudshell_open/gcrunner/terraform
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

Review and deploy:

```sh
terraform plan && terraform apply
```

Type `yes` when prompted. Terraform will create:

| Resource | Purpose |
|---|---|
| Cloud Run service | Webhook receiver and VM orchestrator |
| Service accounts | Minimal IAM for function and runner VMs |
| Artifact Registry | Remote repository for the gcrunner container image |
| Secret Manager secrets | GitHub App credentials storage |
| VPC + Firewall | Network isolation for runner VMs |
| GCS bucket | Build cache storage (if enabled) |

### Save the outputs

```sh
export SETUP_URL=$(terraform output -raw setup_url)
echo "Setup URL: $SETUP_URL"
```

## Set up the GitHub App

Open the setup URL in your browser:

```sh
echo $SETUP_URL
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
