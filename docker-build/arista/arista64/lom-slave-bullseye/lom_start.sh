#! /bin/bash

# Set the build platform environment variable
export DOCKER_BUILD_PLATFORM=arista
export DOCKER_BUILD_OS=linux
export DOCKER_BUILD_ARCH=amd64

export BUILD_CC=/opt/arista/centos7.5-gcc8.4.0-glibc2.17/bin/x86_64-redhat-linux-gcc
export BUILD_CXX=/opt/arista/centos7.5-gcc8.4.0-glibc2.17/bin/x86_64-redhat-linux-g++
# To-Do : Goutham : BUILD_LDFLAGS must have -L/opt/arista/centos7.5-gcc8.4.0-glibc2.17/usr/lib/ instead of other paths. But there are some headers missing
# for certain libraries like libpam. So, we are using the below LDFLAGS for now. This needs to be fixed. It works because our build environemnt is similar to the target environment
# arista64 & also at runtime we are providing the proper libraries on the target environment.
export BUILD_LDFLAGS="-lrt -L/usr/local/lib -L/usr/lib/x86_64-linux-gnu"
# To-Do : Goutham : BUILD_CFLAGS must have -I/opt/arista/centos7.5-gcc8.4.0-glibc2.17/usr/include/ instead of other paths. But there are some headers missing
# for certain libraries like libpam. So, we are using the below CFLAGS for now. This needs to be fixed. It works because our build environemnt is similar to the target environment
# arista64 & also at runtime we are providing the proper libraries on the target environment.
# To-Do : Goutham : Also remove -g from CFLAGS
export BUILD_CFLAGS="-I/usr/include/security -I/usr/include/ -I/usr/include/x86_64-linux-gnu -O2 -g"

export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/local/lib

sudo  /usr/sbin/rsyslogd -n -iNONE &

/bin/bash
exit
