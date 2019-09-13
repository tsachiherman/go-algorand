#!/usr/bin/env bash

# keep script execution on errors
set +e

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"

OS=$(${SCRIPTPATH}/../ostype.sh)
ARCH=$(${SCRIPTPATH}/../archtype.sh)

if [ "${OS}" = "linux" ]; then
    if [[ "${ARCH}" = "arm64" ]]; then
        sudo apt-get -y install sqlite3
        go version
        if [ "$?" != "0" ]; then
            # go is not installed ?
            wget https://dl.google.com/go/go1.12.9.linux-arm64.tar.gz
            if [ "$?" = "0" ]; then
                set -e
                sudo tar -C /usr/local -xzf ./go1.12.9.linux-arm64.tar.gz
                rm -f ./go1.12.9.linux-arm64.tar.gz
                sudo ln -s /usr/local/go/bin/go /usr/local/bin/go
                sudo ln -s /usr/local/go/bin/godoc /usr/local/bin/godoc
                sudo ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt
            else
                echo "Failed to download go"
                exit 1
            fi
        fi
    fi
fi

${SCRIPTPATH}/../configure_dev.sh
