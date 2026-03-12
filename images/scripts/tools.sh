#!/bin/bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

sudo apt-get update

# Packages below are taken directly from the official GitHub runner images
# (actions/runner-images) toolset JSON. This keeps us compatible.

echo "=== Installing vital packages ==="
# From: toolset.apt.vital_packages
sudo apt-get install -y --no-install-recommends \
  bzip2 curl g++ gcc make jq tar unzip wget

echo "=== Installing common packages ==="
# From: toolset.apt.common_packages
sudo apt-get install -y --no-install-recommends \
  autoconf automake dbus dnsutils dpkg dpkg-dev fakeroot \
  fonts-noto-color-emoji gnupg2 iproute2 iputils-ping \
  libyaml-dev libtool libssl-dev libsqlite3-dev locales \
  mercurial openssh-client p7zip-rar pkg-config \
  python-is-python3 rpm texinfo tk tree tzdata upx \
  xvfb xz-utils zsync

echo "=== Installing cmd packages ==="
# From: toolset.apt.cmd_packages
sudo apt-get install -y --no-install-recommends \
  acl aria2 binutils bison brotli coreutils file findutils \
  flex ftp haveged lz4 m4 mediainfo netcat net-tools \
  p7zip-full parallel patchelf pigz pollinate rsync \
  shellcheck sphinxsearch sqlite3 ssh sshpass sudo \
  swig telnet time zip

echo "=== Installing Docker ==="

sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker runner

echo "=== Setting up hostedtoolcache ==="

# This is where actions/setup-node, actions/setup-python, etc. cache tool versions
sudo mkdir -p /opt/hostedtoolcache
sudo chown runner:runner /opt/hostedtoolcache

echo "=== Installing GitHub CLI ==="

curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | \
  sudo tee /etc/apt/sources.list.d/github-cli-stable.list > /dev/null
sudo apt-get update
sudo apt-get install -y gh

echo "=== Tools installation complete ==="
