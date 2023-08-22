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

HOST_SUBDIR="host"
INSTALL_SUBDIR="install

HOST_FILES="etc/systemd/system/device-health.service.d/auto_restart.conf \
    lib/systemd/system/device-health.service \
    usr/bin/device-health.sh"

IMAGE_NAME=docker-device-health
IMAGE_FILE=${IMAGE_NAME}.gz
INSTALL_SCRIPT=LoM-install.sh

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


function forceClean()
{
    pushd /
    for i in ${HOST_FILES}; do
        rm -f $i.${BACK_EXT}
        [[ $? != 0 ]] && { echo "Failed to remove $i.${BACK_EXT}; exit ${ERR_CLEAN}; }
    done
    docker 

    


rm -rf ${WORK_DIR}

pushd fsroot
for i in ${HOST_FILES}
do
    cpFile $i ${HOST_DIR}/$i
done
popd

cpFile target/docker-device-health.gz ${INSTALL_DIR}/docker-device-health.gz

cpFile src/sonic-device-health/vendor/sonic/lom-install.sh ${INSTALL_DIR}/lom-install.sh

tar -cvzf target/LoM-Install.tar.gz ${WORK_DIR}
[[ $? != 0 ]] && { echo "Failed to archive"; exit ${ERR_TAR}; }

echo "all good"
