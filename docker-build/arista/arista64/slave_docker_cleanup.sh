#! /bin/bash

# Run the slave docker with path one level down as root dir.
#
set -x

docker rmi arista64-lom-slave-bullseye-admin:1234
docker rmi arista64-lom-slave-bullseye:1234