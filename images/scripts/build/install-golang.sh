#!/bin/bash -e
################################################################################
##  File:  install-golang.sh
##  Desc:  Install Go (latest stable)
################################################################################

source $HELPER_SCRIPTS/etc-environment.sh
source $HELPER_SCRIPTS/install.sh

GO_VERSION=$(curl -fsSL "https://go.dev/VERSION?m=text" | head -1)
download_url="https://go.dev/dl/${GO_VERSION}.linux-amd64.tar.gz"
archive_path=$(download_with_retry "$download_url")

tar -C /usr/local -xzf "$archive_path"

prepend_etc_environment_path "/usr/local/go/bin"
set_etc_environment_variable "GOROOT" "/usr/local/go"
set_etc_environment_variable "GOPATH" '$HOME/go'
prepend_etc_environment_path '$HOME/go/bin'
