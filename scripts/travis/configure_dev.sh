#!/usr/bin/env bash

# keep script execution on errors
set +e

echo "/scripts/travis/configure_dev.sh called"

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"
OPERATINGSYSTEM=$("${SCRIPTPATH}/../ostype.sh")
ARCH=$("${SCRIPTPATH}/../archtype.sh")

if [[ "${OPERATINGSYSTEM}" == "linux" ]]; then
    if [[ "${ARCH}" == "arm64" ]]; then
        set -e
        sudo apt-get update -y
        sudo apt-get -y install sqlite3 python3-venv libffi-dev libssl-dev
    elif [[ "${ARCH}" == "arm" ]]; then
        sudo sh -c 'echo "CONF_SWAPSIZE=1024" > /etc/dphys-swapfile; dphys-swapfile setup; dphys-swapfile swapon'
        set -e
        sudo apt-get update -y
        sudo apt-get -y install sqlite3
    fi
elif [[ "${OPERATINGSYSTEM}" == "darwin" ]]; then
    # we don't want to upgrade boost if we already have it, as it will try to update
    # other components.
    brew update
    brew tap homebrew/cask
    brew pin boost || true
elif [[ "${OPERATINGSYSTEM}" == "windows" ]]; then
    [[ ! -f C:/tools/msys64/msys2_shell.cmd ]] && rm -rf C:/tools/msys64
    choco uninstall -y mingw
    choco upgrade --no-progress -y msys2
    export msys2='cmd //C RefreshEnv.cmd '
    export msys2+='& set MSYS=winsymlinks:nativestrict '
    export msys2+='& C:\\tools\\msys64\\msys2_shell.cmd -defterm -no-start'
    export mingw64="$msys2 -mingw64 -full-path -here -c "\"\$@"\" --"
    export msys2+=" -msys2 -c "\"\$@"\" --"
    #$msys2 pacman --sync --noconfirm --needed mingw-w64-x86_64-toolchain
    echo "$msys2 pacman" > '"c:\\windows\\pacman.cmd'
    pacman --sync --noconfirm --needed mingw-w64-x86_64-toolchain
    ## Install more MSYS2 packages from https://packages.msys2.org/base here
    taskkill //IM gpg-agent.exe //F  # https://travis-ci.community/t/4967
    export PATH=/C/tools/msys64/mingw64/bin:$PATH
    export MAKE=mingw32-make  # so that Autotools can find it
    shopt -s expand_aliases
fi

"${SCRIPTPATH}/../configure_dev.sh"
exit $?
