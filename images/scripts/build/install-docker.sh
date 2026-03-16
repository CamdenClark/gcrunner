#!/bin/bash -e
################################################################################
##  File:  install-docker.sh
##  Desc:  Install Docker
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh
source $HELPER_SCRIPTS/os.sh

REPO_URL="https://download.docker.com/linux/ubuntu"
GPG_KEY="/usr/share/keyrings/docker.gpg"
REPO_PATH="/etc/apt/sources.list.d/docker.list"
os_codename=$(lsb_release -cs)

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o $GPG_KEY
echo "deb [arch=amd64 signed-by=$GPG_KEY] $REPO_URL ${os_codename} stable" > $REPO_PATH
apt-get update

# Install docker components from toolset
components=$(get_toolset_value '.docker.components[] .package')
for package in $components; do
    version=$(get_toolset_value ".docker.components[] | select(.package == \"$package\") | .version")
    if [[ $version == "latest" ]]; then
        apt-get install --no-install-recommends "$package"
    else
        version_string=$(apt-cache madison "$package" | awk '{ print $3 }' | grep "$version" | grep "$os_codename" | head -1)
        apt-get install --no-install-recommends "${package}=${version_string}"
    fi
done

# Install plugins from GitHub releases
plugins=$(get_toolset_value '.docker.plugins[] .plugin')
for plugin in $plugins; do
    version=$(get_toolset_value ".docker.plugins[] | select(.plugin == \"$plugin\") | .version")
    filter=$(get_toolset_value ".docker.plugins[] | select(.plugin == \"$plugin\") | .asset")
    url=$(resolve_github_release_asset_url "docker/$plugin" "endswith(\"$filter\")" "$version")
    binary_path=$(download_with_retry "$url" "/tmp/docker-$plugin")
    mkdir -pv "/usr/libexec/docker/cli-plugins"
    install "$binary_path" "/usr/libexec/docker/cli-plugins/docker-$plugin"
done

# Fix docker group GID
gid=$(cut -d ":" -f 3 /etc/group | grep "^1..$" | sort -n | tail -n 1 | awk '{ print $1+1 }')
groupmod -g "$gid" docker

# Enable docker.service
systemctl is-active --quiet docker.service || systemctl start docker.service
systemctl is-enabled --quiet docker.service || systemctl enable docker.service

# Docker daemon takes time to come up after installing
sleep 10
docker info

# Cleanup custom repositories
rm $GPG_KEY
rm $REPO_PATH
