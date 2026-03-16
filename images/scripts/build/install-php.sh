#!/bin/bash -e
################################################################################
##  File:  install-php.sh
##  Desc:  Install PHP
##  From:  actions/runner-images (MIT License)
################################################################################

source $HELPER_SCRIPTS/etc-environment.sh
source $HELPER_SCRIPTS/install.sh

php_versions=$(get_toolset_value '.php.versions[]')

for version in $php_versions; do
    echo "Installing PHP $version"
    apt-get install --no-install-recommends \
        php$version \
        php$version-bcmath \
        php$version-bz2 \
        php$version-cgi \
        php$version-cli \
        php$version-common \
        php$version-curl \
        php$version-dev \
        php$version-fpm \
        php$version-gd \
        php$version-gmp \
        php$version-intl \
        php$version-ldap \
        php$version-mbstring \
        php$version-mysql \
        php$version-opcache \
        php$version-pgsql \
        php$version-readline \
        php$version-sqlite3 \
        php$version-xml \
        php$version-xsl \
        php$version-yaml \
        php$version-zip
done

apt-get install --no-install-recommends php-pear

# Install composer
php -r "copy('https://getcomposer.org/installer', 'composer-setup.php');"
php -r "if (hash_file('sha384', 'composer-setup.php') === file_get_contents('https://composer.github.io/installer.sig')) { echo 'Installer verified'; } else { echo 'Installer corrupt'; unlink('composer-setup.php'); } echo PHP_EOL;"
php composer-setup.php
mv composer.phar /usr/bin/composer
php -r "unlink('composer-setup.php');"

prepend_etc_environment_path '$HOME/.config/composer/vendor/bin'
mkdir -p /etc/skel/.composer
