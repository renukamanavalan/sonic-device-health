#! /bin/bash

source $(dirname $0)/common.sh

[[ "${BUILD}" == "" ]] && { echo "Expect BUILD info as env"; exit ${ERR_USAGE}; }
[ ! -d "./target" ] && { echo "Run from buildimage root"; exit ${ERR_USAGE}; }
[ ! -f "./target/docker-device-health.gz" ] && { echo "Build docker"; exit ${ERR_USAGE}; }
[ ! -f "./target/sonic-broadcom.bin" ] && { echo "Build binary image to get service files"; exit ${ERR_USAGE}; }
if [[ "./target/docker-device-health.gz" -nt "./target/sonic-broadcom.bin" ]]; then
      echo "./target/docker-device-health.gz" is newer than "./target/sonic-broadcom.bin"
      echo "re-build image to ensure service files are latest"
      exit ${ERR_USAGE}
fi

WORK_DIR="$(pwd)/tmp/DH-install"
PAYLOAD_DIR="${WORK_DIR}/payload"
HOST_DIR="${PAYLOAD_DIR}/${HOST_SUBDIR}"
INSTALL_DIR="${PAYLOAD_DIR}/${INSTALL_SUBDIR}"
INSTALLER_ARCHIVE=LoM-Install.tar.gz
INSTALLER_SELF_EXTRACT=LoM-Install.bsx

rm -rf ${WORK_DIR}

pushd fsroot-broadcom
for i in ${HOST_FILES}
do
    cpFile $i ${HOST_DIR}/$i
done
popd

echo "1"
INSTALL_FILES="target/${IMAGE_FILE} \
    src/sonic-device-health/install/sonic/${INSTALL_SCRIPT} \
    src/sonic-device-health/install/sonic/${COMMON_SCRIPT}"

for i in ${INSTALL_FILES}; do
    cpFile $i ${INSTALL_DIR}/$(basename $i)
done

TIMESTAMP="$(date +%s)" j2 -o ${INSTALL_DIR}/VERSION -f env src/sonic-device-health/LoM_Version.j2

pushd ${PAYLOAD_DIR}
tar -cvzf ${WORK_DIR}/${INSTALLER_ARCHIVE} .
[[ $? != 0 ]] && { echo "Failed to archive"; exit ${ERR_TAR}; }
popd

cpFile src/sonic-device-health/install/decompress ${WORK_DIR}

pushd ${WORK_DIR}
cat decompress ${INSTALLER_ARCHIVE} > ${INSTALLER_SELF_EXTRACT}
chmod a+x ${INSTALLER_SELF_EXTRACT}
popd

echo "${WORK_DIR}/${INSTALLER_SELF_EXTRACT} is created"
