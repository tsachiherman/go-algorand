#!/usr/bin/env bash

# keep script execution on errors
set +e

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"
OS=$("${SCRIPTPATH}/../ostype.sh")
ARCH=$("${SCRIPTPATH}/../archtype.sh")

if [[ "${OS}" == "linux" ]]; then
    if [[ "${ARCH}" == "arm64" ]]; then
        set -e
        sudo apt-get update -y
        sudo apt-get -y install sqlite3 python3-venv libffi-dev libssl-dev
    elif [[ "${ARCH}" == "arm" ]]; then
        sudo sh -c 'echo "CONF_SWAPSIZE=1024" > /etc/dphys-swapfile; dphys-swapfile setup; dphys-swapfile swapon'
        set -e
        sudo apt-get update -y
        sudo apt-get -y install sqlite3
    elif [[ "${ARCH}" == "amd64" ]]; then
        sudo mv /tmp /old_tmp
        sudo mkdir -p /tmp
        sudo chmod 777 /tmp
        cp -r /old_tmp/ /tmp
        sudo mount -t tmpfs -o rw,size=512M tmpfs /tmp
    fi
elif [[ "${OS}" == "darwin" ]]; then
    # we don't want to upgrade boost if we already have it, as it will try to update
    # other components.
    brew update
    brew tap homebrew/cask
    brew pin boost || true
fi

"${SCRIPTPATH}/../configure_dev.sh"
exit $?
