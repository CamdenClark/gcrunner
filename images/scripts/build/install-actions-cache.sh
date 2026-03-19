#!/bin/bash -e
################################################################################
##  File:  install-actions-cache.sh
##  Desc:  Pre-cache common GitHub Actions to avoid downloading them every job.
##         The runner checks ACTIONS_RUNNER_ACTION_ARCHIVE_CACHE before fetching
##         actions from GitHub. Cached actions are stored as {owner}_{repo}/{SHA}.tar.gz.
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh
source $HELPER_SCRIPTS/etc-environment.sh

CACHE_DIR="/opt/actionarchivecache"
mkdir -p "$CACHE_DIR"
chmod -R 777 "$CACHE_DIR"

# Download the latest action-versions archive from GitHub
archive_url=$(resolve_github_release_asset_url "actions/action-versions" 'endswith("action-versions.tar.gz")' "latest")
archive_path=$(download_with_retry "$archive_url")

# Extract into cache directory
tar -xzf "$archive_path" -C "$CACHE_DIR"
rm -f "$archive_path"

# Set env var so the runner knows where to find cached actions
set_etc_environment_variable "ACTIONS_RUNNER_ACTION_ARCHIVE_CACHE" "$CACHE_DIR"

echo "Actions cache installed to $CACHE_DIR"
ls -la "$CACHE_DIR"
du -sh "$CACHE_DIR"
