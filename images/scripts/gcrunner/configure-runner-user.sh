#!/bin/bash -e
################################################################################
##  File:  configure-runner-user.sh
##  Desc:  Create the runner user for GitHub Actions (gcrunner-specific)
################################################################################

# Create runner user with passwordless sudo
id runner &>/dev/null || useradd -m -s /bin/bash runner
echo "runner ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/runner
chmod 0440 /etc/sudoers.d/runner
