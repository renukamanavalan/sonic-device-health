#! /bin/bash

# Set the build platform environment variable
export DOCKER_BUILD_PLATFORM=arista
export DOCKER_BUILD_OS=linux
export DOCKER_BUILD_ARCH=amd64

export BUILD_CC=/opt/arista/centos7.5-gcc8.4.0-glibc2.17/bin/x86_64-redhat-linux-gcc
export BUILD_CXX=/opt/arista/centos7.5-gcc8.4.0-glibc2.17/bin/x86_64-redhat-linux-g++
export BUILD_LDFLAGS="/usr/local/lib"

sudo  /usr/sbin/rsyslogd -n -iNONE &

/bin/bash
exit
