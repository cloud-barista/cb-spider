#!/bin/bash
source $CBSPIDER_ROOT/setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=nhn-driver-v1.0

go mod download # cb-spider's go.mod and go.sum will be applied.

function build() {
    rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
    CGO_ENABLED=1 go build -buildmode=plugin -o ${DRIVERFILENAME}.so NHNDriver-lib.go || return 1
    chmod +x ${DRIVERFILENAME}.so || return 1
    mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH || return 1
}

build
