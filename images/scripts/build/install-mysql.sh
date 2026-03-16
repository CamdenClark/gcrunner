#!/bin/bash -e
################################################################################
##  File:  install-mysql.sh
##  Desc:  Install MySQL Client and Server
##  From:  actions/runner-images (MIT License)
################################################################################

MYSQL_ROOT_PASSWORD=root
echo "mysql-server mysql-server/root_password password $MYSQL_ROOT_PASSWORD" | debconf-set-selections
echo "mysql-server mysql-server/root_password_again password $MYSQL_ROOT_PASSWORD" | debconf-set-selections

export ACCEPT_EULA=Y

apt-get install mysql-client
apt-get install mysql-server
apt-get install libmysqlclient-dev

# Disable mysql.service
systemctl is-active --quiet mysql.service && systemctl stop mysql.service
systemctl disable mysql.service
