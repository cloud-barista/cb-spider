#!/bin/bash
source ../../../../setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=azure-driver-v1.0

function build() {
    rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
    CGO_ENABLED=1 go build -buildmode=plugin -o ${DRIVERFILENAME}.so AzureDriver-lib.go || return 1
    chmod +x ${DRIVERFILENAME}.so || return 1
    mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH || return 1
}

build
