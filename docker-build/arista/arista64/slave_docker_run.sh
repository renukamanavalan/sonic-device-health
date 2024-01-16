#! /bin/bash

# Run the slave docker with path one level down as root dir.
#
set -x

docker run --rm=true --privileged --init \
    -v "${PWD}:/lom-root" \
    -v "/tmp/docklock:/tmp/docklock"\
    -w "/lom-root" -e "http_proxy=" -e "https_proxy=" -e "no_proxy=" -it \
    arista64-lom-slave-bullseye-admin:1234
