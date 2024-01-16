# Use the latest Debian base image (32-bit)
FROM i386/debian:bullseye-slim

# Copy rsyslog configuration
COPY ["rsyslog.conf", "/etc/"]
COPY ["lom_start.sh", "/usr/local/bin/"]

# Set the build platform label
LABEL build_platform="arista-32"

## Make apt-get non-interactive
ENV DEBIAN_FRONTEND=noninteractive

# Install necessary dependencies for your application
RUN apt-get update && \
    apt-get install -y \
        autoconf \
        automake \
        make \
        sudo \
        libunwind8 \
        wget \
        pkg-config \
        binutils \
        chrpath \
        patchelf \
        tree \
        git \
        libtool \
        vim \
        jq \
        curl \
        unzip \
        zip  \
        python-all \
        python-dev \
        python-setuptools \
        python3 \
        python3-pip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install rsyslog
RUN apt-get update && \
    apt-get install -y --no-install-recommends rsyslog --option=Dpkg::Options::=--force-confdef && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* 

# Install Go
ENV GOLANG_VERSION=1.20.3
RUN wget -O go.tar.gz https://golang.org/dl/go${GOLANG_VERSION}.linux-386.tar.gz && \
    mkdir -p /usr/local/go1.20.3/go && \
    tar -C /usr/local/go1.20.3/go -xzf go.tar.gz --strip-components=1 && \
    rm go.tar.gz

# Add Go binary and pkg-config to PATH
ENV PATH="/usr/local/go1.20.3/bin/:${PATH}"


# Install Arista cross-compiler to /opt/arista
# Refer : https://github.com/aristanetworks/EosSdk/wiki/Build-and-Development-Environment#cross-compiler
#   Cross-Compiler Package	            Use With EOS Target	    installs to
#   arista-fc14.deb	                            < 4.17.0F	    /opt/arista/fc14/
#   arista-fc18-gcc4-9-2.deb	                >= 4.17.0F	    /opt/arista/fc18-gcc4.9.2/
#   arista-fc18-gcc5-4-0.deb	                >= 4.20.0F	    /opt/arista/fc18-gcc5.4.0/ -- Installing this version
#   arista-fc18-gcc6.3.1-glibc2.18.deb	        >= 4.22.0F	    /opt/arista/fc18-gcc6.3.1-glibc2.18/ 
#   arista-centos7-5-gcc6-3-1-glibc2-17.deb	    >= 4.23.0	    /opt/arista/centos7.5-gcc6.3.1-glibc2.17/
#   arista-centos7-5-gcc6-5-0-glibc2-17.deb	    >= 4.23.2	    /opt/arista/centos7.5-gcc6.5.0-glibc2.17/
#   arista-centos7-5-gcc8-4-0-glibc2-17.deb	    >= 4.25.0	    /opt/arista/centos7.5-gcc8.4.0-glibc2.17/
COPY compilers/arista-fc18-gcc5-4-0.deb /tmp/
RUN dpkg -i --force-architecture /tmp/arista-fc18-gcc5-4-0.deb && \
    rm /tmp/arista-fc18-gcc5-4-0.deb

# Install libsodium
RUN wget https://download.libsodium.org/libsodium/releases/LATEST.tar.gz && \
    tar -zxvf LATEST.tar.gz && \
    rm -fr LATEST.tar.gz && \
    cd libsodium-stable && \
    CC=/opt/arista/fc18-gcc5.4.0/bin/i686-pc-linux-gnu-gcc \
    CXX=/opt/arista/fc18-gcc5.4.0/bin/i686-pc-linux-gnu-g++ \
    ./configure && \
    make && \
    make check && \
    make install && \
    cd .. && \
    rm -rf libsodium-stable

# Install libzmq
RUN git clone https://github.com/zeromq/libzmq.git && \
    cd libzmq && \
    ./autogen.sh && \
    CC=/opt/arista/fc18-gcc5.4.0/bin/i686-pc-linux-gnu-gcc \
    CXX=/opt/arista/fc18-gcc5.4.0/bin/i686-pc-linux-gnu-g++ \
    ./configure --with-libsodium=/usr/local/lib/  && \
    make && \
    make install && \
    #make check && \
    cd .. && \
    rm -rf libzmq

# change the rpath of libzmq.so to $ORIGIN(Current directory)
RUN patchelf --set-rpath '$ORIGIN' /usr/local/lib/libzmq.so
