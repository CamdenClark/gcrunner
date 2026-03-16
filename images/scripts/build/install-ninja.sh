#!/bin/bash -e
################################################################################
##  File:  install-ninja.sh
##  Desc:  Install Ninja build system
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

ninja_url=$(resolve_github_release_asset_url "ninja-build/ninja" "endswith(\"ninja-linux.zip\")" "latest")
archive_path=$(download_with_retry "$ninja_url")
unzip -qq "$archive_path" -d /usr/local/bin
