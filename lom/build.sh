#! /bin/bash

# may have to set GO111MODULE=off

# set -x

export GO111MODULE=on
export GO=/usr/local/go/bin/go

# Define output directory
OUTPUT_DIR=$(realpath ./bin)

# Set GOBIN to output directory
export GOBIN=$OUTPUT_DIR

# Install any dependencies specified in go.mod
${GO} mod download -x
${GO} version

install() {
    echo "Installing to $OUTPUT_DIR"
    rm -rf $OUTPUT_DIR
    mkdir -p $OUTPUT_DIR
    ${GO} install ./...
    echo "Build completed successfully."
}

etest() {
    ${GO} test $1 -coverprofile=coverprofile.out  -coverpkg engine -covermode=atomic engine
    if [ $? -ne 0 ]; then
        echo "Failed to run engine test"
        exit -1
    fi
    ${GO} tool cover -html=coverprofile.out -o /tmp/coverage.html
    ls -l coverprofile.out /tmp/coverage.html
    echo "View /tmp/coverage.html in Edge"
}

test() {
    ${GO} test $1 -coverprofile=coverprofile.out  -coverpkg lom/src/lib/lomipc,lom/src/lib/lomcommon -covermode=atomic ./src/lib/lib_test
    echo $?
    if [ $? -ne 0 ]; then
        echo "Failed to run test"
        exit -1
    fi
    ${GO} tool cover -html=coverprofile.out -o /tmp/coverage.html
    ls -l coverprofile.out /tmp/coverage.html
    echo "View /tmp/coverage.html in Edge"
}

clean() {
    echo "run clean"
    rm -rf bin/*
    rm -rf pkg/*
    rm -f coverprofile.out
}

list() {
    tree0
}

cmd="None"

if [ $# -ne 0 ]; then
    cmd=$1
fi  

if [[ "install" == "$cmd"* ]]; then
    install

elif [[ "clean" == "$cmd"* ]]; then
    clean

elif [[ "list" == "$cmd"* ]]; then
    list

elif [[ "test" == "$cmd"* ]]; then
    test ""

elif [[ "vtest" == "$cmd"* ]]; then
    test "-v"

elif [[ "etest" == "$cmd"* ]]; then
    etest ""

elif [[ "xetest" == "$cmd"* ]]; then
    etest "-v"
else
    echo "install / test / clean"
fi

