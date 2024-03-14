#! /bin/bash

# TODO: 
#   Update usr/bin/device-health.sh to mount dirs as desired by plugins
#   Dirs of interest are to be registered in one file by each plugin
#   At build time, it ensures usr/bin/device-health.sh is created such that
#   all these dirs are mounted. May be as RO only.
#
source $(dirname $0)/common.sh

INSTALL_SRC_DIR="$(dirname $0)"
LOM_SRC_DIR="${INSTALL_SRC_DIR}/../.."
LOM_BUILD_DIR="${LOM_SRC_DIR}/lom/build"

WORK_DIR="$(pwd)/tmp/DH-install"
PAYLOAD_DIR="${WORK_DIR}/payload"
HOST_DIR="${PAYLOAD_DIR}/${HOST_SUBDIR}"
INSTALL_DIR="${PAYLOAD_DIR}/${INSTALL_SUBDIR}"
INSTALLER_ARCHIVE=LoM-Install.tar.gz
INSTALLER_SELF_EXTRACT=LoM-Install.bsx
INCLUDE_TEST_ARCHIVE=""

INTEGRATION_TEST_BIN="integration_test_installer.sh"
INTEGRATION_TEST_SRC="${LOM_BUILD_DIR}/${INTEGRATION_TEST_BIN}"
INTEGRATION_TEST_DST="${PAYLOAD_DIR}/${TEST_SUBDIR}/${INTEGRATION_TEST_BIN}"

INSTALL_FILES="target/${IMAGE_FILE} \
    ${INSTALL_SRC_DIR}/${INSTALL_SCRIPT} \
    ${INSTALL_SRC_DIR}/${COMMON_SCRIPT} \
    ${LOM_SRC_DIR}/config/${LOM_VERSION_FILE} \
    ./fsroot-broadcom/etc/sonic/${HOST_VERSION_FILE}"

# Validate ...
[[ ! -d "./target" ]] && { echo "Run from buildimage root"; exit ${ERR_USAGE}; }
for i in ${HOST_FILES}
do
    [[ ! -f fsroot-broadcom/$i ]] && { echo "Missing file fsroot-broadcom/$i"; exit ${ERR_USAGE}; }
done

for i in ${INSTALL_FILES}
do
    [[ ! -f $i ]] && { echo "Missing file $i"; exit ${ERR_USAGE}; }
done

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

for i in ${INSTALL_FILES}; do
    cpFile $i ${INSTALL_DIR}/$(basename $i)
done


if [[ "${INCLUDE_TEST_ARCHIVE}" != "" ]]; then
    cpFile ${INTEGRATION_TEST_SRC} ${INTEGRATION_TEST_DST}
    echo "Copied integration-test code: ${INTEGRATION_TEST_BIN}"
else
    echo "Skip to copy integration-test code: ${INTEGRATION_TEST_BIN}"
fi

pushd ${PAYLOAD_DIR}
tar -cvzf ${WORK_DIR}/${INSTALLER_ARCHIVE} .
[[ $? != 0 ]] && { echo "Failed to archive"; exit ${ERR_TAR}; }
popd

LOM_VERSION_JSON=$(cat ${LOM_SRC_DIR}/config/${LOM_VERSION_FILE} | jq -c | jq -R) \
    HOST_OS_VERSION=$(grep build_version ${INSTALL_DIR}/sonic_version.yml | cut -f2 -d\') \
    j2 -o ${WORK_DIR}/decompress ${INSTALL_SRC_DIR}/../decompress.j2

pushd ${WORK_DIR}
cat decompress ${INSTALLER_ARCHIVE} > ${INSTALLER_SELF_EXTRACT}
chmod a+x ${INSTALLER_SELF_EXTRACT}
popd

echo "${WORK_DIR}/${INSTALLER_SELF_EXTRACT} is created"
cp ${WORK_DIR}/${INSTALLER_SELF_EXTRACT} target/
