#!/bin/bash
set -euo pipefail

echo "=== Installing gcloud CLI ==="

export DEBIAN_FRONTEND=noninteractive

curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | \
  sudo tee /etc/apt/sources.list.d/google-cloud-sdk.list > /dev/null

sudo apt-get update
sudo apt-get install -y google-cloud-cli

echo "=== gcloud CLI installed ==="
