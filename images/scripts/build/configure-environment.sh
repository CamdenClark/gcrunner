#!/bin/bash -e
################################################################################
##  File:  configure-environment.sh
##  Desc:  Configure system and environment
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/etc-environment.sh

# Set ImageVersion and ImageOS env variables
set_etc_environment_variable "ImageVersion" "${IMAGE_VERSION}"
set_etc_environment_variable "ImageOS" "${IMAGE_OS}"

# Set the ACCEPT_EULA variable to Y value
set_etc_environment_variable "ACCEPT_EULA" "Y"

# Create config directory in skel for new users
mkdir -p /etc/skel/.config/configstore
set_etc_environment_variable "XDG_CONFIG_HOME" '$HOME/.config'

# Add localhost alias to ::1 IPv6
sed -i 's/::1 ip6-localhost ip6-loopback/::1     localhost ip6-localhost ip6-loopback/g' /etc/hosts

# Prepare directory and env variable for toolcache
AGENT_TOOLSDIRECTORY=/opt/hostedtoolcache
mkdir -p $AGENT_TOOLSDIRECTORY
set_etc_environment_variable "AGENT_TOOLSDIRECTORY" "${AGENT_TOOLSDIRECTORY}"
set_etc_environment_variable "RUNNER_TOOL_CACHE" "${AGENT_TOOLSDIRECTORY}"
chmod -R 777 $AGENT_TOOLSDIRECTORY

# Kernel tuning
# https://www.elastic.co/guide/en/elasticsearch/reference/current/vm-max-map-count.html
echo 'vm.max_map_count=262144' | tee -a /etc/sysctl.conf

# https://kind.sigs.k8s.io/docs/user/known-issues/#pod-errors-due-to-too-many-open-files
echo 'fs.inotify.max_user_watches=655360' | tee -a /etc/sysctl.conf
echo 'fs.inotify.max_user_instances=1280' | tee -a /etc/sysctl.conf

# https://github.com/actions/runner-images/issues/9491
echo 'vm.mmap_rnd_bits=28' | tee -a /etc/sysctl.conf

# Disable motd updates metadata
sed -i 's/ENABLED=1/ENABLED=0/g' /etc/default/motd-news || true

# Disable man-db auto update
echo "set man-db/auto-update false" | debconf-communicate
dpkg-reconfigure man-db
