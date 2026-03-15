#!/bin/bash
set -euo pipefail

echo "=== Base system setup ==="

export DEBIAN_FRONTEND=noninteractive

sudo apt-get update
sudo apt-get upgrade -y
sudo apt-get install -y \
  build-essential \
  curl \
  wget \
  git \
  jq \
  unzip \
  zip \
  ca-certificates \
  gnupg \
  lsb-release \
  software-properties-common \
  apt-transport-https

# Create runner user (skip if already exists)
id runner &>/dev/null || sudo useradd -m -s /bin/bash runner
echo "runner ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/runner

# Set environment variables for GitHub Actions tool cache compatibility
echo 'AGENT_TOOLSDIRECTORY=/opt/hostedtoolcache' | sudo tee -a /etc/environment
echo 'RUNNER_TOOL_CACHE=/opt/hostedtoolcache' | sudo tee -a /etc/environment

echo "=== Base setup complete ==="
