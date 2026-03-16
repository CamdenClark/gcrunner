#!/bin/bash -e
################################################################################
##  File:  install-ruby.sh
##  Desc:  Install Ruby and ruby gems
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

apt-get install ruby-full

# Install ruby gems from toolset
gems_to_install=$(get_toolset_value ".rubygems[] .name")
if [[ -n "$gems_to_install" ]]; then
    for gem in $gems_to_install; do
        echo "Installing gem $gem"
        gem install --no-document $gem
    done
fi

# Install Ruby requirements
apt-get install libz-dev openssl libssl-dev
