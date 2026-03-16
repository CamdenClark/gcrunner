#!/bin/bash -e
################################################################################
##  File:  install-postgresql.sh
##  Desc:  Install PostgreSQL
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/install.sh

REPO_URL="https://apt.postgresql.org/pub/repos/apt/"

wget -qO - https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor > /usr/share/keyrings/postgresql.gpg
echo "deb [signed-by=/usr/share/keyrings/postgresql.gpg] $REPO_URL $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list

toolset_version=$(get_toolset_value '.postgresql.version')

apt-get update
apt-get install postgresql-$toolset_version
apt-get install libpq-dev

# Disable postgresql.service
systemctl is-active --quiet postgresql.service && systemctl stop postgresql.service
systemctl disable postgresql.service

rm /etc/apt/sources.list.d/pgdg.list
rm /usr/share/keyrings/postgresql.gpg
