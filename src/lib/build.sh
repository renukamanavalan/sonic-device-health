#! /bin/bash

# may have to set GO111MODULE=off

export GOPATH=$(pwd;)

echo $GOPATH

install() {
    echo "run install"
    go install ./...
}

coverage() {
    go test -coverprofile=coverprofile.out  -coverpkg lomipc,lomcommon -covermode=atomic txlib_test
    go tool cover -html=coverprofile.out -o /tmp/coverage.html
    ls -l coverprofile.out /tmp/coverage.html
    echo "View /tmp/coverage.html in Edge"
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

elif [[ "verify" == "$cmd"* ]]; then
    coverage

else
    echo "($cmd) match None"
    echo "install / test / clean"
fi

