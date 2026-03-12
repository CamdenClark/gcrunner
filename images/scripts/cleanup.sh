#!/bin/bash
set -euo pipefail

echo "=== Cleaning up ==="

sudo apt-get clean
sudo apt-get autoremove -y
sudo rm -rf /var/lib/apt/lists/*
sudo rm -rf /tmp/*
sudo rm -rf /var/tmp/*

# Clear logs
sudo journalctl --vacuum-time=1s 2>/dev/null || true
sudo rm -rf /var/log/*.gz /var/log/*.1 /var/log/*.old

echo "=== Cleanup complete ==="
