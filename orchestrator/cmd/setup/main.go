package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

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

func main() {
	port := "3456"
	callbackPath := "/callback"
	redirectURL := fmt.Sprintf("http://localhost:%s%s", port, callbackPath)

	manifest := GitHubAppManifest{
		Name: "gcrunner",
		URL:  "https://github.com/camdenclark/gcrunner",
		HookAttributes: map[string]string{
			"url": "https://example.com/webhook", // placeholder, updated after deploy
		},
		RedirectURL: redirectURL,
		Public:      false,
		DefaultPermissions: map[string]string{
			"actions":        "read",
			"administration": "write",
		},
		DefaultEvents: []string{"workflow_job"},
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		log.Fatalf("Failed to marshal manifest: %v", err)
	}

	mux := http.NewServeMux()

	// Serve the form page that redirects to GitHub
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>gcrunner — Setup</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #f6f8fa; color: #24292f; margin: 0; display: flex; align-items: center; justify-content: center; min-height: 100vh; }
    .card { background: #fff; border: 1px solid #d0d7de; border-radius: 12px; padding: 2.5rem; max-width: 480px; width: 100%%; margin: 1rem; box-shadow: 0 1px 3px rgba(0,0,0,0.04); }
    .logo { font-size: 1.5rem; font-weight: 700; margin-bottom: 0.25rem; }
    .logo span { color: #0969da; }
    .subtitle { color: #656d76; margin: 0 0 1.5rem; font-size: 0.95rem; }
    .btn { display: inline-block; background: #2da44e; color: #fff; border: none; border-radius: 6px; padding: 0.75rem 1.5rem; font-size: 1rem; font-weight: 600; cursor: pointer; transition: background 0.15s; }
    .btn:hover { background: #218838; }
  </style>
</head>
<body>
  <div class="card">
    <div class="logo"><span>gc</span>runner</div>
    <p class="subtitle">Create a GitHub App to connect gcrunner to your repositories.</p>
    <form action="https://github.com/settings/apps/new" method="post">
      <input type="hidden" name="manifest" value='%s'>
      <button type="submit" class="btn">Create GitHub App</button>
    </form>
  </div>
</body>
</html>`, string(manifestJSON))
	})

	// Handle the callback from GitHub after app creation
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code parameter", http.StatusBadRequest)
			return
		}

		// Exchange the code for the app credentials
		resp, err := http.Post(
			fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code),
			"application/json",
			nil,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("GitHub API error: %v", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusCreated {
			http.Error(w, fmt.Sprintf("GitHub returned %d: %s", resp.StatusCode, string(body)), http.StatusInternalServerError)
			return
		}

		var app GitHubAppResponse
		if err := json.Unmarshal(body, &app); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse response: %v", err), http.StatusInternalServerError)
			return
		}

		// Save credentials to .env file
		envContent := fmt.Sprintf(
			"GITHUB_APP_ID=%d\nGITHUB_APP_SLUG=%s\nGITHUB_CLIENT_ID=%s\nGITHUB_CLIENT_SECRET=%s\nGITHUB_WEBHOOK_SECRET=%s\n",
			app.ID, app.Slug, app.ClientID, app.ClientSecret, app.WebhookSecret,
		)
		if err := os.WriteFile(".env", []byte(envContent), 0600); err != nil {
			log.Printf("Failed to write .env: %v", err)
		}

		// Save private key
		if err := os.WriteFile("private-key.pem", []byte(app.PEM), 0600); err != nil {
			log.Printf("Failed to write private-key.pem: %v", err)
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>gcrunner — Setup Complete</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #f6f8fa; color: #24292f; margin: 0; display: flex; align-items: center; justify-content: center; min-height: 100vh; }
    .card { background: #fff; border: 1px solid #d0d7de; border-radius: 12px; padding: 2.5rem; max-width: 480px; width: 100%%; margin: 1rem; box-shadow: 0 1px 3px rgba(0,0,0,0.04); }
    .logo { font-size: 1.5rem; font-weight: 700; margin-bottom: 0.25rem; }
    .logo span { color: #0969da; }
    .success { color: #1a7f37; font-weight: 600; font-size: 1.1rem; margin: 0.5rem 0 1rem; }
    .detail { color: #656d76; font-size: 0.9rem; margin: 0.35rem 0; }
    .detail strong { color: #24292f; }
    .detail a { color: #0969da; }
    .saved { background: #dafbe1; color: #1a7f37; border-radius: 6px; padding: 0.5rem 0.75rem; font-size: 0.85rem; margin: 1rem 0; }
    code { background: #f6f8fa; padding: 0.15rem 0.3rem; border-radius: 3px; font-size: 0.85em; }
    .muted { color: #656d76; font-size: 0.85rem; margin-top: 1rem; }
  </style>
</head>
<body>
  <div class="card">
    <div class="logo"><span>gc</span>runner</div>
    <p class="success">GitHub App created</p>
    <p class="detail"><strong>App:</strong> %s</p>
    <p class="detail"><strong>App ID:</strong> %d</p>
    <p class="detail"><strong>URL:</strong> <a href="%s">%s</a></p>
    <p class="saved">Credentials saved to <code>.env</code> and <code>private-key.pem</code>.</p>
    <p class="muted">You can close this window.</p>
  </div>
</body>
</html>`, app.Name, app.ID, app.HTMLURL, app.HTMLURL)

		log.Printf("GitHub App created: %s (ID: %d)", app.Name, app.ID)
		log.Println("Credentials saved to .env and private-key.pem")
	})

	url := fmt.Sprintf("http://localhost:%s", port)
	log.Printf("Opening %s — create your GitHub App there.", url)

	// Open browser
	go func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		default:
			log.Printf("Please open %s in your browser", url)
			return
		}
		cmd.Run()
	}()

	log.Fatal(http.ListenAndServe(":"+port, mux))
}
