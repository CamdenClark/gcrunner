package function

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
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

func init() {
	functions.HTTP("Webhook", HandleWebhook)
}

// HandleWebhook is the Cloud Function entry point for workflow_job webhooks.
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("WARNING: could not load webhook secret: %v", err)
	}
	if secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifySignature(body, sig, secret) {
			log.Printf("Signature verification failed")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
		log.Printf("Signature verified")
	}

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
