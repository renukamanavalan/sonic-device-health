#! /bin/bash

docker inspect --type image lom-bullseye:1234 &> /dev/null; RET=$?
if [[ ${RET} -ne 0 ]];then
    docker build --no-cache -t lom-bullseye:1234 --build-arg http_proxy= --build-arg https_proxy= --build-arg no_proxy= --build-arg SONIC_VERSION_CACHE= --build-arg SONIC_VERSION_CONTROL_COMPONENTS=none lom-slave-bullseye 2>&1 | tee lom-slave-bullseye/lom-slave-bullseye.log
    docker inspect --type image lom-bullseye:1234 &> /dev/null; RET=$?
    if [[ ${RET} -ne 0 ]];then
        echo "Failed to build base build ..."
        exit -1
    fi
else
    echo "lom-bullseye:1234 is ready"
fi

docker inspect --type image lom-bullseye-admin:1234 &> /dev/null; RET=$?
if [[ ${RET} -ne 0 ]];then
    docker build --no-cache \
        --build-arg user=$(id -un) \
        --build-arg uid=$(id -u) \
        --build-arg guid=$(id -g) \
        --build-arg hostname=${HOSTNAME} \
        -t lom-bullseye-admin:1234 \
        -f lom-slave-bullseye/Dockerfile.user \
        lom-slave-bullseye 2>&1 | tee lom-slave-bullseye/lom-slave-bullseye-admin.log
else
    echo "lom-bullseye-admin:1234 is ready"
fi



