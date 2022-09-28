#!/bin/bash

# Run a container to build drivers with plugin mode.
# The driver repos can be other repos.
# cf) https://github.com/cloud-barista/cb-spider/issues/343
#
# The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
# The CB-Spider Mission is to connect all the clouds with a single interface.
#
#      * Cloud-Barista: https://github.com/cloud-barista
#
# by CB-Spider Team, 2021.04.

# get host go version
GOVERSION=`go version |awk '{print $3}'|sed 's/go//g'`

if [ "$GOVERSION" = "1.19" ]; then
        GOVERSION=${GOVERSION}.0
fi

# for setup current $CBSPIDER_ROOT
source ../../setup.env

# mount volume for driver build script in Container
# mount volume for same Go env. between host and container
# setup HOME env. with host $HOME in container
sudo docker run --rm -it -v $PWD/2.build:$HOME/2.build -v $HOME/go:$HOME/go -v $CBSPIDER_ROOT:$CBSPIDER_ROOT \
	-e GOPATH=$HOME/go -e CBSPIDER_ROOT=$CBSPIDER_ROOT \
	-e  HOME=$HOME -w $HOME --hostname driver-build --name driver-build \
	golang:$GOVERSION /bin/bash

git checkout ../../go.mod ../../go.sum
