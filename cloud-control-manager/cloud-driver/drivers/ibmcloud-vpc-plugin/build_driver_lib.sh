#!/bin/bash
source ../../../../setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=ibmvpc-driver-v1.0

function build() {
    rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
    go build -buildmode=plugin -o ${DRIVERFILENAME}.so IBMCloudDriver-lib.go || return 1
    chmod +x ${DRIVERFILENAME}.so || return 1
    mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH || return 1
}

build
