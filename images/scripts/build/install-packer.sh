#!/bin/bash -e
################################################################################
##  File:  install-packer.sh
##  Desc:  Install HashiCorp Packer
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

download_url=$(curl -fsSL https://api.releases.hashicorp.com/v1/releases/packer/latest | jq -r '.builds[] | select((.arch=="amd64") and (.os=="linux")).url')
archive_path=$(download_with_retry "$download_url")
unzip -o -qq "$archive_path" -d /usr/local/bin
