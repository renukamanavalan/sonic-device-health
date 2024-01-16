#! /bin/bash

# Set the build platform environment variable
export DOCKER_BUILD_PLATFORM=sonic
export DOCKER_BUILD_OS=linux
export DOCKER_BUILD_ARCH=amd64

export BUILD_CC=gcc
export BUILD_CXX=g++

sudo  /usr/sbin/rsyslogd -n -iNONE &
/bin/bash
exit
