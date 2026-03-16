#!/bin/bash -e
################################################################################
##  File:  install-gcc-compilers.sh
##  Desc:  Install GNU C++ compilers
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

versions=$(get_toolset_value '.gcc.versions[]')

for version in ${versions[*]}; do
    echo "Installing $version..."
    apt-get install $version
done
