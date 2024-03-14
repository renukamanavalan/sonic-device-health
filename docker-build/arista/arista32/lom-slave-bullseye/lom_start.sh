#! /bin/bash

# Set the build platform environment variable
export DOCKER_BUILD_PLATFORM=arista
export DOCKER_BUILD_ARCH=386
export DOCKER_BUILD_OS=linux

export BUILD_CC=/opt/arista/fc18-gcc5.4.0/bin/i686-pc-linux-gnu-gcc
export BUILD_CXX=/opt/arista/fc18-gcc5.4.0/bin/i686-pc-linux-gnu-g++
export BUILD_LDFLAGS="/usr/local/lib"

# To-Do : Goutham : BUILD_LDFLAGS must have -L /opt/arista/fc18-gcc5.4.0/usr/lib/ instead of other paths. But there are some headers missing
# for certain libraries like libpam. So, we are using the below LDFLAGS for now. This needs to be fixed. It works because our build environemnt is similar to the target environment
# arista32 & also at runtime we are providing the proper libraries on the target environment.
export BUILD_LDFLAGS="-lrt -L/usr/local/lib -L/usr/lib/i386-linux-gnu/"
# To-Do : Goutham : BUILD_CFLAGS must have -I /opt/arista/fc18-gcc5.4.0/usr/include/ instead of other paths. But there are some headers missing
# for certain libraries like libpam. So, we are using the below CFLAGS for now. This needs to be fixed. It works because our build environemnt is similar to the target environment
# arista32 & also at runtime we are providing the proper libraries on the target environment.
# To-Do : Goutham : Also remove -g from CFLAGS
export BUILD_CFLAGS="-I/usr/include/security -I/usr/include/ -I/usr/include/i386-linux-gnu -O2 -g"

export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/local/lib

sudo  /usr/sbin/rsyslogd -n -iNONE &

/bin/bash
exit
