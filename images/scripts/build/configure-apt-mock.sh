#!/bin/bash -e
################################################################################
##  File:  configure-apt-mock.sh
##  Desc:  Wrap apt/apt-get/apt-key with retry logic for transient failures
##  From:  actions/runner-images (MIT License)
################################################################################

prefix=/usr/local/bin

for real_tool in /usr/bin/apt /usr/bin/apt-get /usr/bin/apt-key; do
    tool=$(basename $real_tool)
    cat >$prefix/$tool <<EOT
#!/bin/sh

i=1
while [ \$i -le 30 ];do
  err=\$(mktemp)
  $real_tool "\$@" 2>\$err

  # no errors, break the loop and continue normal flow
  test -f \$err || break
  cat \$err >&2

  retry=false

  if grep -q 'Could not get lock' \$err;then
    retry=true
  elif grep -q 'Could not get lock /var/lib/apt/lists/lock' \$err;then
    retry=true
  elif grep -q 'Problem renaming the file /var/cache/apt/pkgcache.bin.* to /var/cache/apt/pkgcache.bin' \$err;then
    retry=true
  elif grep -q 'Problem renaming the file /var/cache/apt/srcpkgcache.bin.* to /var/cache/apt/srcpkgcache.bin' \$err;then
    retry=true
  elif grep -q 'Could not open file /var/lib/apt/lists' \$err;then
    retry=true
  elif grep -q 'IPC connect call failed' \$err;then
    retry=true
  elif grep -q 'Temporary failure in name resolution' \$err;then
    retry=true
  elif grep -q 'dpkg frontend is locked by another process' \$err;then
    retry=true
  fi

  rm \$err
  if [ \$retry = false ]; then
    break
  fi

  sleep 5
  echo "...retry \$i"
  i=\$((i + 1))
done
EOT
    chmod +x $prefix/$tool
done
