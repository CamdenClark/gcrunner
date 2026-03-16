#!/bin/bash -e
################################################################################
##  File:  etc-environment.sh
##  Desc:  Helper functions for source and modify /etc/environment
##  From:  actions/runner-images (MIT License)
################################################################################

get_etc_environment_variable() {
    local variable_name=$1
    grep "^${variable_name}=" /etc/environment | sed -E "s%^${variable_name}=\"?([^\"]+)\"?.*$%\1%"
}

add_etc_environment_variable() {
    local variable_name=$1
    local variable_value=$2
    echo "${variable_name}=${variable_value}" | tee -a /etc/environment
}

replace_etc_environment_variable() {
    local variable_name=$1
    local variable_value=$2
    sed -i -e "s%^${variable_name}=.*$%${variable_name}=${variable_value}%" /etc/environment
}

set_etc_environment_variable() {
    local variable_name=$1
    local variable_value=$2

    if grep "^${variable_name}=" /etc/environment > /dev/null; then
        replace_etc_environment_variable $variable_name $variable_value
    else
        add_etc_environment_variable $variable_name $variable_value
    fi
}

prepend_etc_environment_variable() {
    local variable_name=$1
    local element=$2
    existing_value=$(get_etc_environment_variable "${variable_name}")
    set_etc_environment_variable "${variable_name}" "${element}:${existing_value}"
}

append_etc_environment_variable() {
    local variable_name=$1
    local element=$2
    existing_value=$(get_etc_environment_variable "${variable_name}")
    set_etc_environment_variable "${variable_name}" "${existing_value}:${element}"
}

prepend_etc_environment_path() {
    local element=$1
    prepend_etc_environment_variable PATH "${element}"
}

append_etc_environment_path() {
    local element=$1
    append_etc_environment_variable PATH "${element}"
}

reload_etc_environment() {
    eval $(grep -v '^PATH=' /etc/environment | sed -e 's%^%export %')
    etc_path=$(get_etc_environment_variable PATH)
    export PATH="$PATH:$etc_path"
}
