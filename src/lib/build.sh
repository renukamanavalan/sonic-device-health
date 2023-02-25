#! /bin/bash

# may have to set GO111MODULE=off

export GOPATH=$(pwd;)

echo $GOPATH

install() {
    echo "run install"
    go install ./...
}

runtest() {
    echo "run test"
    bin/libtests
    if [ $? -eq 0 ]; then
        echo "TEST SUCCEEDED"
    else
        echo "TEST FAILED"
    fi
}

clean() {
    echo "run clean"
    rm -rf bin/*
    rm -rf pkg/*
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

elif [[ "test" == "$cmd"* ]]; then
    runtest

elif [[ "clean" == "$cmd"* ]]; then
    clean

elif [[ "list" == "$cmd"* ]]; then
    list

else
    echo "($cmd) match None"
    echo "install / test / clean"
fi

