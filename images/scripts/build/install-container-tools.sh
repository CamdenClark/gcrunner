#!/bin/bash -e
################################################################################
##  File:  install-container-tools.sh
##  Desc:  Install container tools: podman, buildah, skopeo
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/os.sh

if ! is_ubuntu22; then
    install_packages=(podman buildah skopeo)
else
    install_packages=(podman=3.4.4+ds1-1ubuntu1 buildah skopeo)
fi

apt-get update
apt-get install ${install_packages[@]}

mkdir -p /etc/containers
printf "[registries.search]\nregistries = ['docker.io', 'quay.io']\n" | tee /etc/containers/registries.conf
