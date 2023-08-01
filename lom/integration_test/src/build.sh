#!/bin/bash

# Exit immediately if any command fails
set -e

GO=/usr/local/go1.20.3/go/bin/go

if [ "$1" == "build" ]; then
    # Remove existing tar file if it exists
    if [ -f "integration_test.tar.gz" ]; then
        rm integration_test.tar.gz
    fi

    # Remove existing binaries in 'integration_test/bin' if they exist
    if [ -d "integration_test/bin" ]; then
        rm -rf integration_test/bin/*
    else
        mkdir -p integration_test/bin
    fi

    # Remove the self extracting installer if it exists
    if [ -f "integration_test_installer.sh" ]; then
        rm integration_test_installer.sh
    fi

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
    cat integration_test/src/self_extracting_installer.sh integration_test.tar.gz > integration_test_installer.sh
    echo "Created self extracting installer 'integration_test_installer.sh'."
    chmod +x integration_test_installer.sh

elif [ "$1" == "clean" ]; then
    # Remove existing tar file if it exists
    if [ -f "integration_test.tar.gz" ]; then
        rm integration_test.tar.gz
        echo "Deleted 'integration_test.tar.gz'."
    fi

    # Remove existing binaries in 'integration_test/bin' if they exist
    if [ -d "integration_test/bin" ]; then
        rm -rf integration_test/bin/*
        echo "Deleted contents of 'integration_test/bin'."
    else
        echo "Directory 'integration_test/bin' is already clean."
    fi

    # Remove the self extracting installer if it exists
    if [ -f "integration_test_installer.sh" ]; then
        rm integration_test_installer.sh
        echo "Deleted 'integration_test_installer.sh'."
    fi

else
    echo "Usage: $0 [build|clean]"
    exit 1
fi
