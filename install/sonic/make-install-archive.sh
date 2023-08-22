#! /bin/bash

source $(dirname $0)/common.sh

[ ! -d "./target" ] && { echo "Run from buildimage root"; exit ${ERR_USAGE}; }
[ ! -f "./target/docker-device-health.gz" ] && { echo "Build docker"; exit ${ERR_USAGE}; }
[ ! -f "./target/sonic-broadcom.bin" ] && { echo "Build binary image to get service files"; exit ${ERR_USAGE}; }
if [[ "./target/docker-device-health.gz" -nt "./target/sonic-broadcom.bin" ]]; then
      echo "./target/docker-device-health.gz" is newer than "./target/sonic-broadcom.bin"
      echo "re-build image to ensure service files are latest"
      exit ${ERR_USAGE}
fi

WORK_DIR="./tmp/DH-install"
HOST_DIR="${WORK_DIR}/${HOST_SUBDIR}"
INSTALL_DIR="${WORK_DIR}/${INSTALL_SUBDIR}"

rm -rf ${WORK_DIR}

pushd fsroot
for i in ${HOST_FILES}
do
    cpFile $i ${HOST_DIR}/$i
done
popd

cpFile target/${IMAGE_FILE} ${INSTALL_DIR}/${IMAGE_FILE}

cpFile src/sonic-device-health/vendor/sonic/${INSTALL_SCRIPT} ${INSTALL_DIR}/${INSTALL_SCRIPT}

tar -cvzf target/LoM-Install.tar.gz ${WORK_DIR}
[[ $? != 0 ]] && { echo "Failed to archive"; exit ${ERR_TAR}; }

echo "all good"
