#! /bin/bash

set -x

docker run --rm=true --privileged --init \
    -v "$(dirname ${PWD}):/lom" \
    -v "/tmp/docklock:/tmp/docklock"\
    -w "/lom-root" -e "http_proxy=" -e "https_proxy=" -e "no_proxy=" -it \
    lom-slave-bullseye-admin:1234 /bin/bash
