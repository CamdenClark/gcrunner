#!/bin/bash -e
################################################################################
##  File:  install-cmake.sh
##  Desc:  Install CMake
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

if command -v cmake; then
    echo "cmake is already installed"
else
    download_url=$(resolve_github_release_asset_url "Kitware/CMake" "endswith(\"inux-x86_64.sh\")" "latest")
    curl -fsSL "${download_url}" -o cmakeinstall.sh

    hash_url=$(resolve_github_release_asset_url "Kitware/CMake" "endswith(\"SHA-256.txt\")" "latest")
    external_hash=$(get_checksum_from_url "$hash_url" "linux-x86_64.sh" "SHA256")
    use_checksum_comparison "cmakeinstall.sh" "$external_hash"

    chmod +x cmakeinstall.sh \
    && ./cmakeinstall.sh --prefix=/usr/local --exclude-subdir \
    && rm cmakeinstall.sh
fi
