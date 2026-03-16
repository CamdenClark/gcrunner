#!/bin/bash -e
################################################################################
##  File:  install-runner-agent.sh
##  Desc:  Install GitHub Actions runner agent (gcrunner-specific)
################################################################################

echo "=== Installing GitHub Actions runner agent ==="

# Determine version
if [ -z "${RUNNER_VERSION:-}" ]; then
    RUNNER_VERSION=$(curl -s https://api.github.com/repos/actions/runner/releases/latest | jq -r '.tag_name' | sed 's/^v//')
fi

echo "Installing runner version: ${RUNNER_VERSION}"

sudo -u runner bash -c "
    cd /home/runner
    curl -sL 'https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz' -o runner.tar.gz
    tar xzf runner.tar.gz
    rm runner.tar.gz
"

# Install runner dependencies (libkrb5, zlib, libssl, libicu, etc.)
/home/runner/bin/installdependencies.sh

echo "=== Runner agent installed (version ${RUNNER_VERSION}) ==="
