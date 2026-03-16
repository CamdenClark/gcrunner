#!/bin/bash -e
################################################################################
##  File:  install-homebrew.sh
##  Desc:  Install Homebrew on Linux
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/etc-environment.sh

# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"

eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"

set_etc_environment_variable HOMEBREW_NO_AUTO_UPDATE 1
set_etc_environment_variable HOMEBREW_CLEANUP_PERIODIC_FULL_DAYS 3650

reload_etc_environment

# Remove gfortran symlink to avoid conflicts with system gfortran
gfortran=$(brew --prefix)/bin/gfortran
if [[ -e $gfortran ]]; then
    rm $gfortran
fi
