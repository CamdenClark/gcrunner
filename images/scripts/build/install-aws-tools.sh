#!/bin/bash -e
################################################################################
##  File:  install-aws-tools.sh
##  Desc:  Install AWS CLI and Session Manager plugin
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

awscliv2_archive_path=$(download_with_retry "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip")
unzip -qq "$awscliv2_archive_path" -d /tmp
/tmp/aws/install -i /usr/local/aws-cli -b /usr/local/bin

smplugin_deb_path=$(download_with_retry "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb")
apt-get install "$smplugin_deb_path"

# Install AWS SAM CLI
aws_sam_cli_archive_name="aws-sam-cli-linux-x86_64.zip"
sam_cli_download_url=$(resolve_github_release_asset_url "aws/aws-sam-cli" "endswith(\"$aws_sam_cli_archive_name\")" "latest")
aws_sam_cli_archive_path=$(download_with_retry "$sam_cli_download_url")

aws_sam_cli_hash=$(get_checksum_from_github_release "aws/aws-sam-cli" "${aws_sam_cli_archive_name}.. " "latest" "SHA256")
use_checksum_comparison "$aws_sam_cli_archive_path" "$aws_sam_cli_hash"

unzip "$aws_sam_cli_archive_path" -d /tmp
/tmp/install
