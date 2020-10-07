#!/usr/bin/env bash

# we want to test this early, since uname migth not be available on windows.
if [[ "${OS}" == *"Windows"* ]]; then
    echo "windows"
fi

UNAME=$(uname)

if [ "${UNAME}" = "Darwin" ]; then
    echo "darwin"
elif [ "${UNAME}" = "Linux" ]; then
    echo "linux"
elif [[ "${UNAME}" == *"MINGW"* ]]; then
    echo "windows"
else
    echo "unsupported"
    exit 1
fi
