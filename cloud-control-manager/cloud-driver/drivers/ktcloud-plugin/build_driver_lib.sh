#!/bin/bash
source $CBSPIDER_ROOT/setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=ktcloud-driver-v1.0

# To be built in the container
go mod download # cb-spider's go.mod and go.sum will be applied.

function build() {
rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
go build -buildmode=plugin -o ${DRIVERFILENAME}.so KtCloudDriver-lib.go || return 1
chmod +x ${DRIVERFILENAME}.so || return 1
mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH || return 1

}

build
