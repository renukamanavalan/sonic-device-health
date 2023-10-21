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

BUILD_VER=$(cat fsroot-broadcom/etc/sonic/sonic_version.yml | grep -e "^build_version" | cut -f2 -d\'| cut -f1 -d .)
[[ "${BUILD_VER}" == "" ]] && { echo "Failed to get build version"; exit ${ERR_USAGE}; }

INSTALL_SRC_DIR="$(dirname $0)"
LOM_SRC_DIR="${INSTALL_SRC_DIR}/../.."

WORK_DIR="$(pwd)/tmp/DH-install"
PAYLOAD_DIR="${WORK_DIR}/payload"
HOST_DIR="${PAYLOAD_DIR}/${HOST_SUBDIR}"
INSTALL_DIR="${PAYLOAD_DIR}/${INSTALL_SUBDIR}"
INSTALLER_ARCHIVE=LoM-Install.tar.gz
INSTALLER_SELF_EXTRACT=LoM-Install.bsx
INCLUDE_TEST_ARCHIVE=""

INTEGRATION_TEST_BIN="integration_test_installer.sh"
INTEGRATION_TEST_SRC="${LOM_SRC_DIR}/lom/integration_test/${INTEGRATION_TEST_BIN}"
INTEGRATION_TEST_DST="${PAYLOAD_DIR}/${TEST_SUBDIR}/${INTEGRATION_TEST_BIN}"

while getopts "t" opt; do
  case ${opt} in
    t )
        INCLUDE_TEST_ARCHIVE="yes"
        echo "Include test archive"
        ;;
   \? )
     echo "Valid options: [-t]"
     exit 1
     ;;
  esac
done


rm -rf ${WORK_DIR}

pushd fsroot-broadcom
for i in ${HOST_FILES}
do
    cpFile $i ${HOST_DIR}/$i
done
popd

INSTALL_FILES="target/${IMAGE_FILE} \
    ${INSTALL_SRC_DIR}/${INSTALL_SCRIPT} \
    ${INSTALL_SRC_DIR}/${COMMON_SCRIPT}"

for i in ${INSTALL_FILES}; do
    cpFile $i ${INSTALL_DIR}/$(basename $i)
done


if [[ "${INCLUDE_TEST_ARCHIVE}" != "" ]]; then
    cpFile ${INTEGRATION_TEST_SRC} ${INTEGRATION_TEST_DST}
    echo "Copied integration-test code: ${INTEGRATION_TEST_BIN}"
else
    echo "Skip to copy integration-test code: ${INTEGRATION_TEST_BIN}"
fi

BUILDVER=${BUILD_VER} TIMESTAMP="$(date +%s)" j2 -o ${INSTALL_DIR}/VERSION -f env src/sonic-device-health/LoM_Version.j2

pushd ${PAYLOAD_DIR}
tar -cvzf ${WORK_DIR}/${INSTALLER_ARCHIVE} .
[[ $? != 0 ]] && { echo "Failed to archive"; exit ${ERR_TAR}; }
popd

j2 -o ${WORK_DIR}/decompress -f json ${INSTALL_SRC_DIR}/../decompress.j2 ${LOM_SRC_DIR}/config/LoM-Version.json

pushd ${WORK_DIR}
cat decompress ${INSTALLER_ARCHIVE} > ${INSTALLER_SELF_EXTRACT}
chmod a+x ${INSTALLER_SELF_EXTRACT}
popd

echo "${WORK_DIR}/${INSTALLER_SELF_EXTRACT} is created"
