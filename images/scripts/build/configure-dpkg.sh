#!/bin/bash -e
################################################################################
##  File:  configure-dpkg.sh
##  Desc:  Configure dpkg for non-interactive installs
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/etc-environment.sh

# Non-interactive frontend
set_etc_environment_variable "DEBIAN_FRONTEND" "noninteractive"

# dpkg: don't ask for confirmation when replacing config files
cat <<EOF >> /etc/apt/apt.conf.d/10dpkg-options
Dpkg::Options {
  "--force-confdef";
  "--force-confold";
}
EOF

# Hide information about packages that are no longer required
cat <<EOF >> /etc/apt/apt.conf.d/10apt-autoremove
APT::Get::AutomaticRemove "0";
APT::Get::HideAutoRemove "1";
EOF
