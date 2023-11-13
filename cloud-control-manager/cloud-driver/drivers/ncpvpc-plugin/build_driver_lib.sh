#!/bin/bash
source $CBSPIDER_ROOT/setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=ncpvpc-driver-v1.0

go mod download # cb-spider's go.mod and go.sum will be applied.

function build() {
    rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
    env GO111MODULE=on go build -buildmode=plugin -o ${DRIVERFILENAME}.so NcpVpcDriver-lib.go || return 1
    chmod +x ${DRIVERFILENAME}.so || return 1
    mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH || return 1
}

build
