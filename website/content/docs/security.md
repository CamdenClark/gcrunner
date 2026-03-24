---
title: "Security"
weight: 5
---

# Security

gcrunner is designed with security as a core requirement. This page covers the potential attack vectors and the controls in place to address each one.

## Webhook Authentication

**Attack vector:** A malicious actor sends a forged webhook to trigger unauthorized VM provisioning.

Every incoming webhook from GitHub is authenticated using HMAC-SHA256. GitHub signs the request body with a shared secret, and gcrunner verifies the `X-Hub-Signature-256` header before processing anything. The comparison uses a constant-time equality check (`hmac.Equal`) to prevent timing attacks. Requests with missing or invalid signatures are rejected with a 401 before any business logic runs.

The webhook secret is stored in Google Secret Manager and never appears in logs, environment variables, or Terraform state.

## Ephemeral, Isolated VMs

**Attack vector:** A compromised job contaminates future jobs or pivots to other workloads.

Each job runs on a freshly provisioned VM that is destroyed when the job finishes. There is no shared state between jobs — no persistent disk, no shared filesystem, and no reused processes. The VM boots from an immutable base image every time.

Self-destruction happens in layers:
1. The runner script deletes the VM on job completion (primary path).
2. The orchestrator force-deletes the VM when it receives the `completed` webhook (secondary path).
3. The boot disk is set to auto-delete so no data persists even if deletion is delayed.

## Just-In-Time Runner Registration

**Attack vector:** A stolen runner registration token is used to register a rogue runner.

gcrunner uses GitHub's JIT (Just-In-Time) runner configuration instead of long-lived registration tokens. A single-use JIT config is generated per job via the GitHub API and passed to the VM through the instance metadata server. The startup script immediately deletes the metadata attribute after reading it, so the config cannot be retrieved from the metadata server after boot.

## Principle of Least Privilege (IAM)

**Attack vector:** A compromised component gains access beyond what it needs.

Three separate service accounts are used, each with only the permissions required for its role:

| Service Account | Role | Permissions |
|---|---|---|
| `gcrunner-function` | Orchestrator (Cloud Run) | Create/delete VMs, read secrets, enqueue tasks |
| `gcrunner-runner` | Attached to runner VMs | Delete itself (self-destruct), access the cache bucket |
| `gcrunner-tasks` | Cloud Tasks invoker | Invoke the Cloud Run `/task/*` endpoints only |

Runner VMs can delete themselves but cannot create or modify other VMs. The orchestrator can attach the runner service account to new VMs but cannot act as it directly.

## Secret Management

**Attack vector:** Credentials are leaked through configuration files, logs, or Terraform state.

All sensitive credentials — the GitHub App ID, RSA private key, webhook secret, and setup token — are stored in Google Secret Manager with automatic replication. They are never written to Terraform state, never logged, and never returned to callers. The orchestrator fetches secrets at runtime using the Secret Manager API.

## Cloud Tasks Authentication

**Attack vector:** An attacker calls the internal `/task/*` endpoints directly to manipulate job state.

The `/task/queued` and `/task/completed` endpoints are only invoked by Cloud Tasks, not directly by GitHub. Cloud Tasks authenticates each request with an OIDC token signed by the `gcrunner-tasks` service account. The Cloud Run IAM policy enforces this — only that service account (and the unauthenticated `/webhook` path) is granted the `run.invoker` role.

Task names are derived from the job ID, which deduplicates retries and prevents a replayed webhook from provisioning multiple VMs for the same job.

## GitHub App Scopes

**Attack vector:** A compromised GitHub App token is used to access repositories or perform unintended actions.

The GitHub App is configured with minimal scopes:
- `actions: read` — to read workflow job metadata
- `administration: write` — to register and deregister runners

It receives only `workflow_job` webhook events. It is not listed publicly on the GitHub Marketplace.

## Setup Endpoint Protection

**Attack vector:** An unauthorized user completes the setup flow and takes control of the GitHub App credentials.

The `/setup` endpoint requires a pre-shared setup token to initiate the OAuth flow. This token is a cryptographically random value stored in Secret Manager and provided to the operator at deploy time. The OAuth state parameter is HMAC-signed with a nonce and includes a 1-hour TTL, providing CSRF protection throughout the GitHub App creation flow.
