#!/bin/bash

GO=/usr/local/go1.20.3/go/bin/go

# Run 'make all' in the current directory
make all-silent

# Check if the build succeeded
if [ $? -eq 0 ]; then
    echo "Build succeeded."

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

    # Copy new files from 'build/bin' to 'integration_test/bin'
    cp -R build/bin/* integration_test/bin/
    echo "Copied new files to 'integration_test/bin'."

    # Navigate to linkcrc_mocker directory and build
    pushd src/plugins/plugins_files/sonic/plugin_integration_tests/linkcrc_mocker
    $GO build .
    popd

    # Copy the linkcrc_mocker binary to 'integration_test/bin'
    cp src/plugins/plugins_files/sonic/plugin_integration_tests/linkcrc_mocker/linkcrc_mocker integration_test/bin/
    echo "Copied linkcrc_mocker to 'integration_test/bin'."

    # Make all binaries in 'integration_test/bin' executable
    chmod +x integration_test/bin/*
    echo "Made all binaries in 'integration_test/bin' executable."

    # Create a tar archive of 'integration_test'
    #tar -czvf integration_test.tar.gz integration_test
    #echo "Created tar archive 'integration_test.tar.gz'."
else
    echo "Build failed."
fi