#! /bin/bash

ERR_MKDIR=-1
ERR_CP=-2
ERR_TAR=-3
ERR_USAGE=-4
ERR_BACKUP=-5
ERR_CLEAN=-6
ERR_INSTALL_CODE=-7
ERR_ROLLBACK=-8
ERR_RUNTIME=-9
ERR_USAGE=-10
ERR_TEST=-11
ERR_RESTART=-12
ERR_DB=-13

HOST_SUBDIR="host"
INSTALL_SUBDIR="install"
TEST_SUBDIR="test"

HOST_FILES="\
    lib/systemd/system/device-health.service \
    usr/bin/device-health.sh"

IMAGE_NAME="docker-device-health"
IMAGE_FILE="${IMAGE_NAME}.gz"
INSTALL_SCRIPT="LoM-install.sh"
COMMON_SCRIPT="common.sh"
LOM_VERSION_FILE="LoM-Version.json"
HOST_VERSION_FILE="sonic_version.yml"

BACK_EXT="bak"

function makeDir()
{
    mkdir -p $1
    [[ $? != 0 ]] && { echo "Failed to make dir"; exit ${ERR_MKDIR}; }
}


function cpFile()
{
    makeDir $(dirname $2)
    cp $1 $2
    [[ $? != 0 ]] && { echo "Failed to copy file $1 to $2"; exit ${ERR_CP}; }
}


