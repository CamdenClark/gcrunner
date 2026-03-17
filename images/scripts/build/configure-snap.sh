#!/bin/bash -e
################################################################################
##  File:  configure-snap.sh
##  Desc:  Hold snap auto-refresh to prevent mid-build updates
##  Note:  Snap can auto-update packages during a workflow run, which causes
##         flaky failures when binaries change out from under a running job.
##         This holds auto-refresh for 60 days. After that, snaps will resume
##         normal update behavior.
################################################################################

source $HELPER_SCRIPTS/etc-environment.sh

# Hold snap auto-refresh for 60 days to prevent mid-build updates
# See: https://snapcraft.io/docs/managing-updates
snap set system refresh.hold="$(date --date='60 days' +%Y-%m-%dT%H:%M:%S%:z)"

# Add /snap/bin to PATH if not already present
prepend_etc_environment_path "/snap/bin"
