#!/bin/bash -e
################################################################################
##  File:  install-actions-cache.sh
##  Desc:  Download latest release from actions/action-versions
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh
source $HELPER_SCRIPTS/etc-environment.sh

# Prepare directory for ACTIONS_RUNNER_ACTION_ARCHIVE_CACHE
ACTION_ARCHIVE_CACHE_DIR=/opt/actionarchivecache
mkdir -p $ACTION_ARCHIVE_CACHE_DIR
chmod -R 777 $ACTION_ARCHIVE_CACHE_DIR
set_etc_environment_variable "ACTIONS_RUNNER_ACTION_ARCHIVE_CACHE" "${ACTION_ARCHIVE_CACHE_DIR}"

# Download latest release from github.com/actions/action-versions
download_url=$(resolve_github_release_asset_url "actions/action-versions" "endswith(\"action-versions.tar.gz\")" "latest")
archive_path=$(download_with_retry "$download_url")
tar -xzf "$archive_path" -C $ACTION_ARCHIVE_CACHE_DIR
