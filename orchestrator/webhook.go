package function

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var (
	smClient     *secretmanager.Client
	smClientOnce sync.Once
	projectID    string
)

func getSecretManagerClient(ctx context.Context) (*secretmanager.Client, error) {
	var initErr error
	smClientOnce.Do(func() {
		smClient, initErr = secretmanager.NewClient(ctx)
		projectID = os.Getenv("GCP_PROJECT")
		if projectID == "" {
			projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}
	})
	return smClient, initErr
}

func getSecret(ctx context.Context, name string) (string, error) {
	client, err := getSecretManagerClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secret manager client: %w", err)
	}

	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name),
	})
	if err != nil {
		return "", fmt.Errorf("failed to access secret %s: %w", name, err)
	}

	return strings.TrimSpace(string(result.Payload.Data)), nil
}

// writeSecret creates a secret if it doesn't exist and adds a new version with the given value.
func writeSecret(ctx context.Context, name, value string) error {
	client, err := getSecretManagerClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secret manager client: %w", err)
	}

	secretName := fmt.Sprintf("projects/%s/secrets/%s", projectID, name)

	// Create secret if it doesn't exist
	_, err = client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", projectID),
		SecretId: name,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		return fmt.Errorf("failed to create secret %s: %w", name, err)
	}

	// Add new version
	_, err = client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretName,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add secret version for %s: %w", name, err)
	}

	return nil
}

// HandleWebhook is the HTTP entry point that routes between
// setup pages and webhook handling.
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/setup":
		handleSetup(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/setup/callback":
		handleSetupCallback(w, r)
	default:
		handleWebhook(w, r)
	}
}

// generateState creates an HMAC-signed state token containing a timestamp.
// The state is verified on callback to prevent CSRF without needing server-side storage.
func generateState(setupToken string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	nonceHex := hex.EncodeToString(nonce)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	payload := nonceHex + "." + ts
	mac := hmac.New(sha256.New, []byte(setupToken))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}

// verifyState checks that a state token is valid and not expired (1 hour TTL).
func verifyState(state, setupToken string) bool {
	parts := strings.SplitN(state, ".", 3)
	if len(parts) != 3 {
		return false
	}
	payload := parts[0] + "." + parts[1]
	sig, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(setupToken))
	mac.Write([]byte(payload))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return false
	}
	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	return time.Since(time.Unix(ts, 0)) < 1*time.Hour
}

func handleSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate setup token
	setupToken, err := getSecret(ctx, "gcrunner-setup-token")
	if err != nil || setupToken == "" {
		log.Printf("ERROR: could not load setup token: %v", err)
		http.Error(w, "setup not available", http.StatusInternalServerError)
		return
	}
	providedToken := r.URL.Query().Get("token")
	if !hmac.Equal([]byte(providedToken), []byte(setupToken)) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Generate CSRF state parameter
	state, err := generateState(setupToken)
	if err != nil {
		log.Printf("ERROR: could not generate state: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	functionURL := fmt.Sprintf("https://%s", r.Host)

	manifest := GitHubAppManifest{
		Name: "gcrunner",
		URL:  "https://github.com/camdenclark/gcrunner",
		HookAttributes: map[string]string{
			"url": functionURL,
		},
		RedirectURL: functionURL + "/setup/callback",
		Public:      false,
		DefaultPermissions: map[string]string{
			"actions":        "read",
			"administration": "write",
		},
		DefaultEvents: []string{"workflow_job"},
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		http.Error(w, "failed to marshal manifest", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<body>
  <h1>gcrunner — GitHub App Setup</h1>
  <p>Click the button below to create a GitHub App for gcrunner.</p>
  <form action="https://github.com/settings/apps/new" method="post">
    <input type="hidden" name="manifest" value='%s'>
    <input type="hidden" name="state" value="%s">
    <button type="submit" style="font-size:1.2em;padding:10px 20px;cursor:pointer;">
      Create GitHub App
    </button>
  </form>
</body>
</html>`, string(manifestJSON), state)
}

func handleSetupCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	// Verify CSRF state parameter
	state := r.URL.Query().Get("state")
	setupToken, err := getSecret(ctx, "gcrunner-setup-token")
	if err != nil || setupToken == "" {
		log.Printf("ERROR: could not load setup token for state verification: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if state == "" || !verifyState(state, setupToken) {
		log.Printf("Setup callback: invalid or expired state parameter")
		http.Error(w, "invalid or expired state", http.StatusForbidden)
		return
	}

	// Exchange the code for app credentials
	resp, err := http.Post(
		fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code),
		"application/json",
		nil,
	)
	if err != nil {
		log.Printf("ERROR: GitHub API error: %v", err)
		http.Error(w, "failed to contact GitHub API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERROR: failed to read GitHub response: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusCreated {
		log.Printf("ERROR: GitHub returned %d: %s", resp.StatusCode, string(body))
		http.Error(w, "GitHub did not accept the app manifest", http.StatusInternalServerError)
		return
	}

	var app GitHubAppResponse
	if err := json.Unmarshal(body, &app); err != nil {
		log.Printf("ERROR: failed to parse GitHub response: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Write credentials to Secret Manager
	secrets := map[string]string{
		"gcrunner-app-id":          fmt.Sprintf("%d", app.ID),
		"gcrunner-private-key":     app.PEM,
		"gcrunner-webhook-secret":  app.WebhookSecret,
	}
	for name, value := range secrets {
		if err := writeSecret(ctx, name, value); err != nil {
			log.Printf("ERROR writing secret %s: %v", name, err)
			http.Error(w, "failed to save credentials", http.StatusInternalServerError)
			return
		}
		log.Printf("Wrote secret %s", name)
	}

	installURL := fmt.Sprintf("%s/installations/new", app.HTMLURL)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<body>
  <h1>GitHub App created!</h1>
  <p><strong>App:</strong> %s</p>
  <p><strong>App ID:</strong> %d</p>
  <p>Credentials have been saved to Secret Manager.</p>
  <p><a href="%s" style="font-size:1.2em;">Install the app on your repositories →</a></p>
</body>
</html>`, app.Name, app.ID, installURL)

	log.Printf("GitHub App created: %s (ID: %d)", app.Name, app.ID)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	log.Printf("Received %s request from %s, event: %s", r.Method, r.RemoteAddr, r.Header.Get("X-GitHub-Event"))

	// Verify HMAC signature using secret from Secret Manager
	secret, err := getSecret(ctx, "gcrunner-webhook-secret")
	if err != nil {
		log.Printf("ERROR: could not load webhook secret: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if secret == "" {
		log.Printf("ERROR: webhook secret is empty")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	sig := r.Header.Get("X-Hub-Signature-256")
	if !verifySignature(body, sig, secret) {
		log.Printf("Signature verification failed")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}
	log.Printf("Signature verified")

	event := r.Header.Get("X-GitHub-Event")
	if event != "workflow_job" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "ignored event: %s", event)
		return
	}

	var payload WorkflowJobEvent
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	switch payload.Action {
	case "queued":
		if err := handleQueued(ctx, payload); err != nil {
			log.Printf("ERROR handling queued: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case "completed":
		if err := handleCompleted(ctx, payload); err != nil {
			log.Printf("ERROR handling completed: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case "in_progress":
		// no-op for MVP
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok")
}

func handleQueued(ctx context.Context, event WorkflowJobEvent) error {
	labels := parseLabels(event.WorkflowJob.Labels)
	if labels == nil {
		log.Printf("Job %d: not a gcrunner job, skipping", event.WorkflowJob.ID)
		return nil
	}

	log.Printf("Job %d: creating VM with labels %+v", event.WorkflowJob.ID, labels)
	return createRunnerVM(ctx, event, labels)
}

func handleCompleted(ctx context.Context, event WorkflowJobEvent) error {
	labels := parseLabels(event.WorkflowJob.Labels)
	if labels == nil {
		return nil
	}

	instanceName := fmt.Sprintf("gcrunner-%d-%d", event.WorkflowJob.RunID, event.WorkflowJob.ID)
	log.Printf("Job %d: completed, deleting VM %s", event.WorkflowJob.ID, instanceName)
	return deleteRunnerVM(ctx, instanceName)
}

func verifySignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sig, err := hex.DecodeString(signature[7:])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)
	return hmac.Equal(sig, expected)
}

// WorkflowJobEvent represents a GitHub workflow_job webhook payload.
type WorkflowJobEvent struct {
	Action      string      `json:"action"`
	WorkflowJob WorkflowJob `json:"workflow_job"`
	Repository  Repository  `json:"repository"`
}

type WorkflowJob struct {
	ID     int64    `json:"id"`
	RunID  int64    `json:"run_id"`
	Labels []string `json:"labels"`
}

type Repository struct {
	FullName string         `json:"full_name"`
	Owner    RepositoryOwner `json:"owner"`
	Name     string         `json:"name"`
}

type RepositoryOwner struct {
	Login string `json:"login"`
}

// GitHubAppManifest is the manifest sent to GitHub to create a new App.
type GitHubAppManifest struct {
	Name               string            `json:"name"`
	URL                string            `json:"url"`
	HookAttributes     map[string]string `json:"hook_attributes"`
	RedirectURL        string            `json:"redirect_url"`
	Public             bool              `json:"public"`
	DefaultPermissions map[string]string `json:"default_permissions"`
	DefaultEvents      []string          `json:"default_events"`
}

// GitHubAppResponse is the response from the manifest conversion endpoint.
type GitHubAppResponse struct {
	ID            int    `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	WebhookSecret string `json:"webhook_secret"`
	PEM           string `json:"pem"`
	HTMLURL       string `json:"html_url"`
}
