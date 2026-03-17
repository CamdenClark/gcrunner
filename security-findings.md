# Security Findings

## 1. Webhook Auth Bypass When Secret Manager Is Unavailable (CRITICAL)

**File:** `orchestrator/webhook.go:232-245`

If `getSecret()` fails, the error is logged but the request proceeds unauthenticated. The `if secret != ""` guard means a Secret Manager outage or quota exhaustion silently disables all webhook signature verification.

An attacker who can trigger or wait for this condition can send forged webhooks to spin up VMs or exfiltrate JIT configs.

**Fix:** Hard-fail the handler — return 500 if the secret can't be loaded.

**Status:** Fix implemented in this PR.

## 2. Setup Flow Has No Auth or CSRF Protection (HIGH)

**File:** `orchestrator/webhook.go:107-214`

`/setup` and `/setup/callback` are publicly accessible (Cloud Run has `allUsers` as `roles/run.invoker`). Anyone who discovers the Cloud Run URL can initiate GitHub App creation, and the callback overwrites Secret Manager credentials with no verification.

No CSRF `state` parameter is used, so an attacker could also trick an admin into completing a callback that swaps credentials.

**Fix:** Two layers:
1. **Setup token gate** — Terraform generates a random token stored in Secret Manager. `/setup` only renders if `?token=<value>` is provided, checked server-side only (never sent to GitHub).
2. **CSRF `state` parameter** — Random nonce generated per setup attempt, stored server-side, validated on callback.

**Status:** Fix implemented in this PR.

## 3. JIT Config Visible in VM Metadata (MEDIUM)

**File:** `orchestrator/vm.go:15-57`

The JIT config (which contains GitHub runner credentials) is embedded in the startup script via `fmt.Sprintf`. This is visible to anyone who can call `gcloud compute instances describe` on the VM.

In practice, access to `compute.instances.get` on the project already implies significant GCP access. But if runner SA permissions are ever broadened, a compromised job could read another VM's metadata to get its JIT config.

**Mitigation options:**
- Pass JIT config via instance metadata with a separate key and retrieve it once at boot, then delete the metadata entry
- Use Secret Manager to pass the JIT config (runner SA already has secret accessor)

## 4. Runner SA Can Delete Any VM in the Project (MEDIUM)

**File:** `terraform/iam.tf:40-45`

The runner service account has `roles/compute.instanceAdmin.v1` at the project level. A compromised job could delete other runner VMs, not just itself.

**Fix:** Use a custom IAM role scoped to `compute.instances.delete` with a condition on the `gcrunner=true` label, or use a separate cleanup SA.

## 5. No HTTP Client Timeouts on GitHub API Calls (LOW)

**File:** `orchestrator/github.go`

All GitHub API calls use `http.DefaultClient` with no timeout. A hung GitHub API response ties up the Cloud Run instance. Not directly exploitable but an availability concern.

**Fix:** Use `&http.Client{Timeout: 30 * time.Second}` or set `context.WithTimeout` on requests.
