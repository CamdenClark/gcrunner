# gcrunner — Technical Specification

> **Status:** Draft v0.1
> **Last updated:** 2026-03-12

## Overview

gcrunner is an open-source, drop-in replacement for GitHub-hosted Actions runners on Google Cloud Platform. It provisions ephemeral Compute Engine VMs on demand, runs the job, and self-destructs — giving teams full `actions/runner-images` compatibility at 80%+ cost savings, especially for startups sitting on $100k–$350k in GCP credits.

---

## High-Level Architecture

```
┌──────────────┐     webhook_job      ┌─────────────────────┐
│    GitHub     │ ──────────────────►  │   Cloud Function    │
│   Actions     │   (queued/          │   (Go, gen2)        │
│   Workflow    │    in_progress/     │                     │
└──────────────┘    completed)        │  1. Parse labels    │
                                      │  2. Map to GCE spec │
                                      │  3. Create VM       │
                                      └────────┬────────────┘
                                               │
                                               ▼
                                      ┌─────────────────────┐
                                      │  Compute Engine VM  │
                                      │  (ephemeral)        │
                                      │                     │
                                      │  1. Boot (~25s)     │
                                      │  2. Register runner │
                                      │  3. Execute job     │
                                      │  4. Self-destruct   │
                                      └─────────────────────┘
```

### Request Flow

1. A GitHub Actions workflow triggers a `workflow_job` webhook event (action: `queued`).
2. The webhook hits a Cloud Function (HTTP-triggered, Go).
3. The Cloud Function authenticates the webhook (HMAC-SHA256), parses the `runs-on` labels, and translates them into a Compute Engine VM spec.
4. The Cloud Function calls the Compute Engine API to create an ephemeral VM with a startup script that:
   - Downloads and configures the GitHub Actions runner agent
   - Registers as an ephemeral (single-use) runner for the target repository
   - Signals readiness back to GitHub
5. GitHub dispatches the job to the new runner.
6. On job completion, the VM executes a shutdown script that deregisters the runner and deletes itself.

### Completed / In-Progress Hooks

- `in_progress`: No-op in MVP. Future use for monitoring/metrics.
- `completed`: Safety net. If the VM failed to self-destruct, the Cloud Function force-deletes it by instance name (derived from run ID + job ID).

### Infrastructure (Single Terraform Apply)

All infrastructure is deployed via a single `terraform apply`:

| Resource | Purpose |
|---|---|
| Cloud Function (gen2) | Webhook receiver, VM orchestrator |
| Service Account | Minimal IAM: `compute.instanceAdmin.v1`, `iam.serviceAccountUser` on runner SA |
| Runner Service Account | Attached to VMs. Access to GCS cache bucket, Artifact Registry (future) |
| VPC + Subnet | Dedicated network for runner VMs (or use existing) |
| Firewall Rules | Egress-only by default, no ingress |
| GCS Bucket | Build cache storage |
| Secret Manager Secret | GitHub App private key + webhook secret |

---

## Job Label Configuration

Labels are the primary interface for configuring runners. Following runs-on's proven model, all configuration lives directly in the `runs-on:` field of the workflow file — no external config files needed for basic usage.

### Syntax

```yaml
jobs:
  build:
    runs-on: gcrunner=${{ github.run_id }}/machine=n2-standard-4
```

The prefix `gcrunner=<run_id>` identifies the job as targeting gcrunner. Everything after the prefix is a `/`-separated list of `key=value` labels.

Array syntax is also supported but single-string syntax is recommended to prevent GitHub's label-subset matching from causing runner collisions between jobs (same issue runs-on documents).

### MVP Labels

In MVP, the user specifies an exact GCE machine type. No family prefix expansion, no CPU/RAM ranges, no multi-machine selection logic. The Cloud Function takes the machine type string and passes it straight to the Compute Engine API.

| Label | Default | Description |
|---|---|---|
| `machine` | `e2-medium` | Full GCE machine type name (e.g. `n2-standard-4`, `c3-standard-8`, `e2-medium`). Passed directly to the Compute Engine API. |
| `spot` | `true` | Use Spot VMs. `true` (default), `false` for on-demand. If spot capacity is unavailable, automatically falls back to on-demand. |
| `disk` | `50gb` | Boot disk size (e.g. `50gb`, `100gb`, `200gb`). |
| `image` | `ubuntu24-full-x64` | Runner image. Maps to a GCE image family. |

That's it. Four labels.

**Examples:**

```yaml
# Simplest — e2-medium spot VM, 50gb disk (all defaults)
runs-on: gcrunner=${{ github.run_id }}

# Specify machine type
runs-on: gcrunner=${{ github.run_id }}/machine=n2-standard-4

# Bigger machine, more disk
runs-on: gcrunner=${{ github.run_id }}/machine=c3-standard-8/disk=200gb

# Production deploy — no spot, can't risk preemption
runs-on: gcrunner=${{ github.run_id }}/machine=n2-standard-2/spot=false
```

**How spot works in MVP:**

1. By default, all VMs are Spot (`spot=true`). GCP Spot VMs are 60–91% cheaper than on-demand.
2. If GCP cannot fulfill the Spot request (capacity error), gcrunner automatically retries the same machine type as on-demand.
3. If a running Spot VM gets preempted mid-job, GCP gives a 30-second warning. The runner agent catches SIGTERM and the job fails. The user can re-run manually.
4. Zone selection: try each zone in the configured region sequentially until one succeeds.

**Built-in images (MVP):**

| Image ID | Base | Arch | Description |
|---|---|---|---|
| `ubuntu24-full-x64` | Ubuntu 24.04 | x86_64 | Full runner image (matches `ubuntu-latest`) |
| `ubuntu22-full-x64` | Ubuntu 22.04 | x86_64 | Legacy compat |

### Post-MVP Labels

These labels are deferred to post-MVP to keep the initial implementation simple.

| Label | Description |
|---|---|
| `cpu` | Number of vCPUs. Supports ranges (`2+8`). Combined with `machine` family prefix for auto-selection. |
| `ram` | Memory in GB. Supports ranges (`8+32`). |
| `machine` (family prefix) | Accept partial names (`n2`, `c3`) or multi-family (`n2+c3`). Requires smart machine type selection logic. |
| `disk-type` | Disk type: `pd-balanced`, `pd-ssd`, `pd-standard`. |
| `local-ssd` | Attach local SSDs for scratch space. |
| `spot-fallback` | Explicit control over fallback behavior (vs MVP's always-fallback). |
| `retry` | Auto-retry on spot preemption (`when-preempted`). |
| `max-price` | Budget guardrail — max hourly price. |
| `zone` | Pin to specific zone(s). |
| `timeout` | Max job runtime in minutes. |
| `debug` | Pause before first step for SSH debugging. |
| `extras` | Feature flags: `gcs-cache`, `docker-cache`. |
| `runner` | Reference a reusable runner profile from `.github/gcrunner.yml`. |
| `image` (ARM) | ARM images (`ubuntu24-full-arm64`) for `t2a` family. |

---

## VM Lifecycle

### Boot Sequence (~25s target)

1. **VM Created** — Compute Engine API returns, VM starts booting.
2. **Startup Script Runs:**
   a. Set hostname to `gcrunner-{run_id}-{job_id}`
   b. Pull runner config from instance metadata (repo, token, labels)
   c. Download GitHub Actions runner agent (cached in GCE image for fast boot)
   d. Configure runner as ephemeral (`--ephemeral` flag)
   e. Register with GitHub (`config.sh`)
   f. Start runner (`run.sh`)
3. **Runner Ready** — GitHub dispatches the job.

### Shutdown Sequence

1. Job completes → runner agent exits.
2. Shutdown script runs:
   a. Upload any cache artifacts to GCS (if `gcs-cache` enabled)
   b. Deregister runner from GitHub (safety, ephemeral runners auto-deregister)
   c. VM calls `gcloud compute instances delete --quiet` on itself
3. If self-delete fails, the Cloud Function's `completed` webhook handler force-deletes after a grace period.

### Zombie VM Protection

Multiple layers prevent orphaned VMs:

1. **Self-destruct on job completion** (primary)
2. **`completed` webhook force-delete** (secondary)
3. **VM `timeout` label** — max runtime, VM auto-terminates via metadata-based watchdog
4. **Terraform-managed Cloud Scheduler** — periodic sweep that deletes any gcrunner VM older than `max_vm_age` (default: 6 hours)

---

## Security Model

- **Webhook authentication**: HMAC-SHA256 verification of all incoming webhooks.
- **Minimal IAM**: Cloud Function service account has only `compute.instanceAdmin.v1` on the runner subnet, `iam.serviceAccountUser` on the runner SA. Runner SA has only GCS access (cache bucket) and `compute.instances.delete` on itself.
- **Ephemeral runners**: Every VM handles exactly one job. No shared state between jobs. No persistent disk reuse.
- **No ingress**: Runner VMs have no public IP by default (configurable). All communication is outbound to GitHub and GCS.
- **GitHub App scoping**: The GitHub App only needs `actions:read` and `administration:write` (for runner registration). No code access.
- **Secrets in Secret Manager**: GitHub App private key and webhook secret stored in Secret Manager, not in Terraform state.

---

## MVP Scope (Weeks 1–4)

### Weeks 1–2: Core Loop
- [ ] Cloud Function: webhook receiver + HMAC auth
- [ ] Label parser: `gcrunner=` prefix detection, key=value parsing (4 labels: `machine`, `spot`, `disk`, `image`)
- [ ] VM orchestrator: take exact machine type string → Compute Engine API call
- [ ] Spot with auto-fallback: try spot, if capacity error → retry on-demand
- [ ] Zone selection: try zones in configured region sequentially until one succeeds
- [ ] Startup script: runner agent download, config, register, run
- [ ] Shutdown script: deregister + self-delete
- [ ] Terraform module: all infrastructure
- [ ] Zombie sweep (Cloud Scheduler)
- [ ] Built-in image: `ubuntu24-full-x64` (based on `actions/runner-images`)

### Weeks 3–4: Polish + Cache
- [ ] GCS build cache (`extras=gcs-cache`)
- [ ] `ubuntu22-full-x64` image
- [ ] Documentation site
- [ ] Example workflows
- [ ] Monitoring basics (Cloud Logging structured logs)

### Post-MVP
- Machine family prefix expansion (`machine=n2` → auto-select size)
- CPU/RAM range labels (`cpu=4+8`, `ram=16+32`)
- Multi-family selection (`machine=n2+c3`)
- ARM runners (`t2a` + `ubuntu24-full-arm64`)
- Custom runner definitions (`.github/gcrunner.yml`)
- Spot preemption retry (`retry=when-preempted`)
- `max-price` budget guardrails
- `debug` mode with SSH
- Windows runners
- GPU/TPU support (`g2`, `a3` families)
- Warm pools (pre-provisioned VMs)
- Docker layer caching (Artifact Registry)
- Local SSD auto-mount + RAID
- CLI setup tool (`gcrunner init`)
- Monitoring dashboard (Cloud Monitoring)
- Cost reporting
- Org-level runner definitions
- Zone intelligence (preemption tracking, smart zone ranking)
