#! /bin/bash

# Run the slave docker with path one level down as root dir.
#
set -x

pushd $(dirname $0)
# Builds if needed, else no-op
echo "check for image: pwd=${PWD}"
./slave_docker_build.sh
popd

docker run --rm=true --privileged --init \
    -v "${PWD}:/lom-root" \
    -v "/tmp/docklock:/tmp/docklock"\
    -w "/lom-root" -e "http_proxy=" -e "https_proxy=" -e "no_proxy=" -it \
    lom-slave-bullseye-admin:1234
