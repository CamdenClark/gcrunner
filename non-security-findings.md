# Non-Security Findings

## 1. No Idempotency for VM Creation

**File:** `orchestrator/vm.go:43-79`

If GitHub delivers a `queued` webhook twice (which GitHub documents as possible), the second attempt tries to create a VM with the same name and fails. This leaves the first VM running correctly but produces error logs.

**Fix:** Check if the instance already exists before calling `Insert`, or handle the `AlreadyExists` error gracefully.

## 2. Silent Failure on VM Deletion

**File:** `orchestrator/vm.go:175-193`

The delete loop swallows all errors — permission errors, quota issues, API failures all get `continue`. The caller has no way to know if deletion actually succeeded vs the VM not existing vs a real failure.

**Fix:** Distinguish "not found" errors (expected) from other errors (should be surfaced).

## 3. Hardcoded Zone Suffixes

**File:** `orchestrator/vm.go:65`

```go
zones := []string{region + "-a", region + "-b", region + "-c", region + "-f"}
```

Not all GCP regions have zones a, b, c, f. Some regions have different zone layouts. This will silently fail in those regions until it hits a valid zone.

**Fix:** Query available zones via the Compute API, or make zones configurable.

## 4. No Validation of Webhook Payload Fields

**File:** `orchestrator/webhook.go:254-258`

JSON unmarshaling doesn't validate that required fields are present. Missing `WorkflowJob.ID`, `Repository.Owner.Login`, etc. would produce zero-value strings/ints, leading to VMs named `gcrunner-0-0` or API calls with empty parameters.

**Fix:** Validate required fields after unmarshaling, return 400 if missing.

## 5. Error Messages Leak Internal Details

**File:** `orchestrator/webhook.go:163, 175, 194`

```go
http.Error(w, fmt.Sprintf("GitHub API error: %v", err), http.StatusInternalServerError)
```

Error responses include raw error messages from GitHub API and Secret Manager, which could reveal internal infrastructure details to callers.

**Fix:** Return generic error messages to the client, log the details server-side only.

## 6. Container Image Uses `latest` Tag

**File:** `terraform/function.tf:28`

The Cloud Run service references the container image by `latest` tag rather than a digest. This means a compromised registry or build pipeline could silently swap the running image.

**Fix:** Pin to a specific image digest (`sha256:...`).
