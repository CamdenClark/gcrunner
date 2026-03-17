# CLAUDE.md

## Project Overview

gcrunner is a self-hosted GitHub Actions runner platform on Google Cloud. It receives `workflow_job` webhooks via a Cloud Run service, provisions ephemeral Compute Engine VMs, runs the job, and self-destructs the VM when done.

## Repository Structure

- `orchestrator/` — Go service: webhook receiver, label parser, VM orchestrator (Cloud Run)
- `cache-server/` — Go service: GCS-backed build cache server
- `terraform/` — Terraform config for all GCP infrastructure
- `website/` — Hugo documentation site (hosted at gcrunner.com)
- `images/` — Packer/scripts for building runner VM images

## Languages & Tools

- **Go** for backend services (`orchestrator`, `cache-server`)
- **Terraform** for infrastructure
- **Hugo** for the documentation website

## Development Commands

```sh
# Orchestrator
cd orchestrator && go build ./...
cd orchestrator && go test ./...

# Cache server
cd cache-server && go build ./...
cd cache-server && go test ./...

# Terraform
cd terraform && terraform init && terraform validate

# Website
cd website && hugo server
```

## Key Concepts

- All runner config lives in the workflow `runs-on:` field using `/`-separated `key=value` labels
- VMs are ephemeral — one job per VM, self-destruct on completion
- Spot VMs are used by default with automatic fallback to on-demand
- Webhook authentication uses HMAC-SHA256
