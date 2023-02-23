#! /bin/bash


export GOPATH=$(pwd;)

echo $GOPATH

install() {
    echo "run install"
    go install ./...
    tree0
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

else
    echo "($cmd) match None"
    echo "install / test / clean"
fi

