#!/bin/bash -e
################################################################################
##  File:  install-yq.sh
##  Desc:  Install yq - YAML, JSON and XML processor
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

yq_url=$(resolve_github_release_asset_url "mikefarah/yq" "endswith(\"yq_linux_amd64\")" "latest")
binary_path=$(download_with_retry "${yq_url}")

hash_url=$(resolve_github_release_asset_url "mikefarah/yq" "endswith(\"checksums\")" "latest")
external_hash=$(get_checksum_from_url "${hash_url}" "yq_linux_amd64 " "SHA256" "true" " " "19")
use_checksum_comparison "$binary_path" "$external_hash"

install "$binary_path" /usr/bin/yq
