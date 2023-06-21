#!/bin/bash

# Run 'make all' in the current directory
make all-silent

# Check if the build succeeded
if [ $? -eq 0 ]; then
    echo "Build succeeded."

    # Remove existing tar file if it exists
    if [ -f "int_test.tar.gz" ]; then
        rm int_test.tar.gz
    fi

    # Remove existing binaries in 'int_test/bin' if they exist
    if [ -d "int_test/bin" ]; then
        rm -rf int_test/bin/*
    else
        mkdir -p int_test/bin
    fi

    # Copy new files from 'build/bin' to 'int_test/bin'
    cp -R build/bin/* int_test/bin/
    echo "Copied new files to 'int_test/bin'."

    # Make all binaries in 'int_test/bin' executable
    chmod +x int_test/bin/*
    echo "Made all binaries in 'int_test/bin' executable."

    # Create a tar archive of 'int_test'
    tar -czvf int_test.tar.gz int_test
    echo "Created tar archive 'int_test.tar.gz'."
else
    echo "Build failed."
fi

