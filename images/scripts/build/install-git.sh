#!/bin/bash -e
################################################################################
##  File:  install-git.sh
##  Desc:  Install Git and Git-FTP
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

GIT_REPO="ppa:git-core/ppa"

# Install git from PPA
add-apt-repository $GIT_REPO -y
apt-get update
apt-get install git

# Git version 2.35.2+ security fix - mark all directories as safe
cat <<EOF >> /etc/gitconfig
[safe]
        directory = *
EOF

# Install git-ftp
apt-get install git-ftp

# Remove source repo
add-apt-repository --remove $GIT_REPO

# Add well-known SSH host keys to known_hosts
ssh-keyscan -t rsa,ecdsa,ed25519 github.com >> /etc/ssh/ssh_known_hosts
ssh-keyscan -t rsa ssh.dev.azure.com >> /etc/ssh/ssh_known_hosts
