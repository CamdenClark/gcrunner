#!/bin/bash
set -euo pipefail

echo "=== Installing GitHub Actions runner agent ==="

cd /home/runner

# Determine version
if [ -z "${RUNNER_VERSION:-}" ]; then
  RUNNER_VERSION=$(curl -s https://api.github.com/repos/actions/runner/releases/latest | jq -r '.tag_name' | sed 's/^v//')
fi

echo "Installing runner version: ${RUNNER_VERSION}"

sudo -u runner curl -sL \
  "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz" \
  -o runner.tar.gz

sudo -u runner tar xzf runner.tar.gz
sudo rm runner.tar.gz

# Install runner dependencies (libkrb5, zlib, libssl, libicu, etc.)
sudo ./bin/installdependencies.sh

echo "=== Runner agent installed (version ${RUNNER_VERSION}) ==="
