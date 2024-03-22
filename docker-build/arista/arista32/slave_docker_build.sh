#! /bin/bash

# Builds slave docker containers to give a arista build environment for LoM.
# This way every dev uses exactly same env for building.
# This ensure right versions of dependent packages are resolved.
#

# Required by docker build
mkdir -p lom-slave-bullseye/vcache

docker inspect --type image arista32-lom-slave-bullseye:1234 &> /dev/null; RET=$?
if [[ ${RET} -ne 0 ]];then
    #docker build --no-cache -t arista-lom-slave-bullseye:1234 --build-arg http_proxy= --build-arg https_proxy= --build-arg no_proxy= --build-arg SONIC_VERSION_CACHE= --build-arg SONIC_VERSION_CONTROL_COMPONENTS=none arista-lom-slave-bullseye 2>&1 | tee arista-lom-slave-bullseye/arista-lom-slave-bullseye.log
    docker build --no-cache \
            -t arista32-lom-slave-bullseye:1234 \
             lom-slave-bullseye 2>&1 | tee lom-slave-bullseye/arista32-lom-slave-bullseye.log
    docker inspect --type image arista32-lom-slave-bullseye:1234 &> /dev/null; RET=$?
    if [[ ${RET} -ne 0 ]];then
        echo "Failed to build base build ..."
        exit -1
    fi
else
    echo "arista32-lom-slave-bullseye:1234 is ready"
fi

docker inspect --type image arista32-lom-slave-bullseye-admin:1234 &> /dev/null; RET=$?
if [[ ${RET} -ne 0 ]];then
    docker build --no-cache \
        --build-arg user=$(id -un) \
        --build-arg uid=$(id -u) \
        --build-arg guid=$(id -g) \
        --build-arg hostname=${HOSTNAME} \
        -t arista32-lom-slave-bullseye-admin:1234 \
        -f lom-slave-bullseye/Dockerfile.user \
        lom-slave-bullseye 2>&1 | tee lom-slave-bullseye/arista32-lom-slave-bullseye-admin.log
else
    echo "arista32-lom-slave-bullseye-admin:1234 is ready"
fi



