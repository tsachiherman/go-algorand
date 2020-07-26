#!/usr/bin/env bash

# build_test.sh - Performs a build & test on the branch
#
# Syntax:   build_test.sh
#
# Usage:    Can be used by either Travis or an ephermal build machine
#
# Examples: scripts/travis/build_test.sh
set -e

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"
df -h
echo "TMPDIR=$TMPDIR"
mkdir -p ~/tmp
chmod 777 ~/tmp
sudo mount -F tmpfs -o size=256 swap ~/tmp
export TMPDIR=~/tmp
echo "TMPDIR=$TMPDIR"
sudo mount -v

if [ "${USER}" = "travis" ]; then
    # we're running on a travis machine
    "${SCRIPTPATH}/build.sh" --make_debug
    "${SCRIPTPATH}/travis_wait.sh" 90 "${SCRIPTPATH}/test.sh"
else
    # we're running on an ephermal build machine
    "${SCRIPTPATH}/build.sh" --make_debug
    "${SCRIPTPATH}/test.sh"
fi
