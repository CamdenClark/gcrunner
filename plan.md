# Webhook Queue Design: Cloud Tasks

## Overview

Replace synchronous webhook processing with an async queue using Google Cloud Tasks. The webhook handler validates the signature and immediately returns 200, then enqueues the work. Cloud Tasks delivers the task back to Cloud Run at a dedicated path, where the actual VM creation/deletion happens. Cloud Tasks handles retries automatically.

## Architecture

```
GitHub Webhook → Cloud Run /webhook → validate + enqueue → 200 OK
                 Cloud Tasks → Cloud Run /task/queued    → JIT config + create VM
                 Cloud Tasks → Cloud Run /task/completed → delete VM
```

## Changes

### 1. Terraform (`terraform/`)

- **`main.tf`**: Add `cloudtasks.googleapis.com` to the enabled APIs list
- **New `queue.tf`**:
  - Create `google_cloud_tasks_queue` named `gcrunner-webhook` in the same region
  - Retry config: max 5 attempts, min backoff 5s, max backoff 300s, doubling
  - Rate limit: 10 dispatches/sec (prevents GCE API hammering)
- **`iam.tf`**:
  - Grant the function SA `roles/cloudtasks.enqueuer` so it can create tasks
  - Create a dedicated SA for Cloud Tasks to invoke Cloud Run (`gcrunner-tasks`) with `roles/run.invoker`
- **`function.tf`**: Add `CLOUD_TASKS_QUEUE` and `CLOUD_RUN_URL` env vars to the Cloud Run service

### 2. New file: `orchestrator/queue.go`

- Cloud Tasks client (lazy-init like the secret manager client)
- `enqueueTask(ctx, path string, payload []byte)` function:
  - Creates an HTTP task targeting `{CLOUD_RUN_URL}{path}`
  - Sets OIDC token with the tasks SA email for auth
  - Uses `X-CloudTasks-Retry-Count` header (auto-set by Cloud Tasks) for observability
  - Task name derived from job ID for deduplication (prevents double-enqueue on GitHub webhook retries)

### 3. Modify `orchestrator/webhook.go`

- `handleWebhook`: After validation + parsing, instead of calling `handleQueued`/`handleCompleted` directly, call `enqueueTask` with the relevant path and the raw payload body
- New `HandleTask` function registered at `/task/` prefix:
  - Validates the request comes from Cloud Tasks (check `X-CloudTasks-TaskName` header presence — Cloud Run IAM handles auth)
  - Routes `/task/queued` → existing `handleQueued` logic
  - Routes `/task/completed` → existing `handleCompleted` logic
- `handleQueued` and `handleCompleted` remain unchanged — they do the same work, just invoked from a different path now

### 4. Modify `orchestrator/cmd/server/main.go`

- Add route: `http.HandleFunc("/task/", orchestrator.HandleTask)`

## What stays the same

- Signature verification still happens in the webhook handler (before enqueue)
- `handleQueued`, `handleCompleted`, `createRunnerVM`, `deleteRunnerVM` logic is untouched
- Deterministic instance naming (`gcrunner-{runID}-{jobID}`) still provides idempotency
- Setup routes (`/setup`, `/setup/callback`) unchanged

## Retry behavior

Cloud Tasks retry config means:
- If VM creation fails (all zones exhausted), the task fails and Cloud Tasks retries after 5s, then 10s, 20s, etc.
- Max 5 attempts before the task is dropped
- This replaces relying on GitHub's webhook retry (which has a 10s timeout and much longer retry intervals)

## New dependency

- `cloud.google.com/go/cloudtasks` added to `go.mod`
