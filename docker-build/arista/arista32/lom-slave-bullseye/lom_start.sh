#! /bin/bash

# Set the build platform environment variable
export DOCKER_BUILD_PLATFORM=arista
export DOCKER_BUILD_ARCH=386
export DOCKER_BUILD_OS=linux

export BUILD_CC=/opt/arista/fc18-gcc5.4.0/bin/i686-pc-linux-gnu-gcc
export BUILD_CXX=/opt/arista/fc18-gcc5.4.0/bin/i686-pc-linux-gnu-g++
export BUILD_LDFLAGS="/usr/local/lib"

sudo  /usr/sbin/rsyslogd -n -iNONE &

/bin/bash
exit
