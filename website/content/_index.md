---
title: "gcrunner"
type: docs
---

# gcrunner

Self-hosted GitHub Actions runners on your Google Cloud. Drop-in replacement for GitHub-hosted runners at 80%+ cost savings.

## How it works

gcrunner receives GitHub Actions `workflow_job` webhooks via a Cloud Run service, provisions an ephemeral Compute Engine VM with your requested specs, runs the job, and self-destructs the VM when done.

```
GitHub Actions → Webhook → Cloud Run (Go) → Compute Engine VM → Self-destruct
```

Completely open source and within your infrastructure.

## Get started

<a href="https://shell.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https%3A%2F%2Fgithub.com%2Fcamdenclark%2Fgcrunner&cloudshell_tutorial=tutorial.md" target="_blank" rel="noopener noreferrer">
  <img src="/open-btn.svg" alt="Open in Cloud Shell">
</a>

Deploy in ~15 minutes with the guided Cloud Shell tutorial, or follow the [installation instructions]({{< relref "/docs/installation" >}}) for manual setup.

## Attribution

gcrunner was heavily inspired by [runs-on](https://runs-on.com) and was mostly written to see if it was possible to build a similar solution on Google Cloud.
