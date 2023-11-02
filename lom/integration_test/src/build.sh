#!/bin/bash

# Exit immediately if any command fails
set -e

GO=/usr/local/go1.20.3/go/bin/go

function rmFileOrDir() {
    rm -rf $1
    if [[ $? -ne 0 ]]; then
        echo "Error removing $1"
        exit -1
    fi
    echo "Removed $1"
}


function clean() {
    for i in ${TEST_BIN} "integration_test.tar.gz" "integration_test/bin" 
    do
        rmFileOrDir ${i}
    done

}


if [ "$1" == "build" ]; then
    if [[ $# -ne 2 ]]; then
        echo "Need target location"
        exit 1
    fi
    TEST_BIN="$2"
    clean

    mkdir -p $(dirname ${TEST_BIN})
    mkdir -p integration_test/bin

    # Copy new files from 'build/bin' to 'integration_test/bin'
    cp -R build/bin/* integration_test/bin/
    echo "Copied new files to 'integration_test/bin'."

    # Check if linkcrc_mocker binary exists
    if [ ! -f "build/test/linkcrc_mocker" ]; then
        echo "Error: linkcrc_mocker binary not found in 'build/test' directory."
        exit 1
    fi

    # Copy the linkcrc_mocker binary to 'integration_test/bin'
    cp build/test/linkcrc_mocker integration_test/bin/
    echo "Copied linkcrc_mocker to 'integration_test/bin'."

    # Navigate to utils directory and build
    pushd integration_test/src/utils
    if ! $GO build -o command_listener; then
        echo "Error: Failed to build command_listener."
        popd
        exit 1
    fi
    popd

    # Check if the utils binary exists
    if [ ! -f "integration_test/src/utils/command_listener" ]; then
        echo "Error: command_listener binary not found in 'integration_test/src/utils' directory."
        exit 1
    fi

    # Move the utils binary to 'integration_test/bin'
    mv integration_test/src/utils/command_listener integration_test/bin/
    echo "Copied command_listener to 'integration_test/bin'."

    # Make all binaries in 'integration_test/bin' executable
    chmod +x integration_test/bin/*
    echo "Made all binaries in 'integration_test/bin' executable."

    # Create a tar archive of 'integration_test'
    tar -czvf integration_test.tar.gz integration_test
    echo "Created tar archive 'integration_test.tar.gz'."

    # create a self extracting installer
    mkdir -p $(dirname ${TEST_BIN})
    cat integration_test/src/self_extracting_installer.sh integration_test.tar.gz > ${TEST_BIN}
    echo "Created self extracting installer 'integration_test_installer.sh'."
    chmod +x ${TEST_BIN}

elif [ "$1" == "clean" ]; then
    clean
else
    echo "Usage: $0 [build|clean] <bin file>"
    exit 1
fi
