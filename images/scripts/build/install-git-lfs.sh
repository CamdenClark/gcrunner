#!/bin/bash -e
################################################################################
##  File:  install-git-lfs.sh
##  Desc:  Install Git LFS
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

GIT_LFS_REPO="https://packagecloud.io/install/repositories/github/git-lfs"

curl -fsSL $GIT_LFS_REPO/script.deb.sh | bash
apt-get install git-lfs

rm -f /etc/apt/sources.list.d/github_git-lfs.list
