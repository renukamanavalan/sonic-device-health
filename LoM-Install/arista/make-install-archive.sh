#!/bin/bash

# Exit immediately if any command fails
set -e

# Configuration
VENDOR="arista"
BUILD_DIR="./build"
CONFIG_DIR="$BUILD_DIR/config/$VENDOR"
SYSTEM_LIBS_DIR="/usr/local/lib"
ROOT_INSTALL_DIR="../LoM-Install"
ROOT_INSTALL_SCRIPT="LoM-install.sh"

INSTALLER_DIR="$BUILD_DIR/installer"
LIBRARY_DIR="$INSTALLER_DIR/libs"
INSTALLER_BIN_DIR="$INSTALLER_DIR/install"
INSTALLER_CONFIG_DIR="$INSTALLER_DIR/config"
INSTALLER_ARCHIVE="LoM-Install.tar.gz"
INSTALLER_SELF_EXTRACT="LoM-Install.bsx"
ERR_TAR=1

# Check command-line arguments
if [ $# -ne 1 ]; then
    echo "Usage: $0 [build|clean]"
    exit 1
fi

# Build the installer
if [ "$1" == "build" ]; then
    # Create necessary directories
    mkdir -p "$BUILD_DIR"
    rm -rf "$INSTALLER_DIR"
    mkdir -p "$INSTALLER_DIR"
    mkdir -p "$INSTALLER_BIN_DIR"
    mkdir -p "$INSTALLER_CONFIG_DIR"
    mkdir -p "$LIBRARY_DIR"

    # Copy ZMQ libraries to LIBRARY_DIR directory
    find /usr/local/lib/ -name 'libzmq.so*' -exec cp {} "$LIBRARY_DIR/" \;
    find /usr/local/lib/ -name 'libsodium.so*' -exec cp {} "$LIBRARY_DIR/" \;

    # change the rpath of the binaries depend on zmq libs to point to the libs directory. This is needed for engine to find the zmq libraries
    patchelf --set-rpath '$ORIGIN/../libs' $BUILD_DIR/bin/LoMEngine
    patchelf --set-rpath '$ORIGIN/../libs' $BUILD_DIR/bin/LoMCli   

    # Copy binaries to INSTALL_DIR directory
    cp -R $BUILD_DIR/bin/* "$INSTALLER_BIN_DIR/"

    # Copy config files to INSTALLER_CONFIG_DIR directory
    cp -R $CONFIG_DIR/* "$INSTALLER_CONFIG_DIR/"

    # Copy necessary scripts to INSTALLER_DIR directory
    cp "$ROOT_INSTALL_DIR/$VENDOR/$ROOT_INSTALL_SCRIPT" "$INSTALLER_BIN_DIR/"
    cp "$ROOT_INSTALL_DIR/$VENDOR/do-install.py" "$INSTALLER_BIN_DIR/"
    cp "$ROOT_INSTALL_DIR/$VENDOR/arista_eapi_helper.py" "$INSTALLER_BIN_DIR/"
    cp "$ROOT_INSTALL_DIR/$VENDOR/arista_cli_helper.py" "$INSTALLER_BIN_DIR/"
    cp "$ROOT_INSTALL_DIR/$VENDOR/common.py" "$INSTALLER_BIN_DIR/"
    #cp "$ROOT_INSTALL_DIR/$VENDOR/deprecated_do-install_native.py" "$INSTALLER_BIN_DIR/"
    #cp "$ROOT_INSTALL_DIR/$VENDOR/deprecated_helpers_native.py" "$INSTALLER_BIN_DIR/"
    #cp "$ROOT_INSTALL_DIR/$VENDOR/deprecated_arista_eapi_helper_pyeapi.py" "$INSTALLER_BIN_DIR/"

    # Make all binaries in INSTALLER_BIN_DIR executable
    chmod +x "$INSTALLER_BIN_DIR"/*

    # Create the installer archive
    tar -czvf "${BUILD_DIR}/${INSTALLER_ARCHIVE}" -C "${INSTALLER_DIR}" .
    if [ $? -ne 0 ]; then
        echo "Failed to create the archive."
        exit "$ERR_TAR"
    fi

    # Create the self-extracting installer
    pushd "$BUILD_DIR"
    cat "./LoM-Install/decompress.j2" "$INSTALLER_ARCHIVE" > "$INSTALLER_SELF_EXTRACT"
    chmod a+x "$INSTALLER_SELF_EXTRACT"
    popd

    echo "${BUILD_DIR}/${INSTALLER_SELF_EXTRACT} is created"
    
# Clean up
elif [ "$1" == "clean" ]; then
    rm -rf "$INSTALLER_DIR"
    rm -rf "$BUILD_DIR/$INSTALLER_ARCHIVE"
    rm -rf "$BUILD_DIR/$INSTALLER_SELF_EXTRACT"

else
    echo "Usage: $0 [build|clean]"
    exit 1
fi

# Exit gracefully
exit 0
