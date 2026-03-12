#!/bin/bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

# Install packages that match the official runner images (actions/runner-images)
# Vital + common packages from their toolset
echo "=== Installing essential packages ==="

sudo apt-get update
sudo apt-get install -y \
  acl \
  aria2 \
  autoconf \
  automake \
  binutils \
  bison \
  brotli \
  bzip2 \
  coreutils \
  dnsutils \
  dpkg-dev \
  fakeroot \
  flex \
  ftp \
  gnupg2 \
  iproute2 \
  iputils-ping \
  libc++-dev \
  libc++abi-dev \
  libffi-dev \
  libgbm-dev \
  libgconf-2-4 \
  libicu-dev \
  libkrb5-3 \
  liblttng-ust1 \
  libssl-dev \
  libtool \
  libyaml-dev \
  locales \
  lz4 \
  m4 \
  mediainfo \
  mercurial \
  net-tools \
  netcat-openbsd \
  openssh-client \
  p7zip-full \
  parallel \
  patchelf \
  pigz \
  pkg-config \
  python-is-python3 \
  rpm \
  rsync \
  shellcheck \
  sqlite3 \
  ssh \
  sudo \
  swig \
  tar \
  telnet \
  texinfo \
  tk \
  tzdata \
  xvfb \
  xz-utils \
  zip \
  zlib1g-dev \
  zsync

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
