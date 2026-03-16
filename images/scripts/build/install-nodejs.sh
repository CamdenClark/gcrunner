#!/bin/bash -e
################################################################################
##  File:  install-nodejs.sh
##  Desc:  Install Node.js LTS and related tooling
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

# Install default Node.js
default_version=$(get_toolset_value '.node.default')
curl -fsSL https://raw.githubusercontent.com/tj/n/master/bin/n -o ~/n
bash ~/n $default_version

# Install node modules
node_modules=$(get_toolset_value '.node_modules[].name')
npm install -g $node_modules

# Fix global modules installation as regular user
chmod -R 777 /usr/local/lib/node_modules
chmod -R 777 /usr/local/bin

rm -rf ~/n
