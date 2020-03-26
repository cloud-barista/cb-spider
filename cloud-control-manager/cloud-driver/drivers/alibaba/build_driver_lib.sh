#!/bin/bash
source ../../../../setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=ali-driver-v1.0

rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
go build -buildmode=plugin -o ${DRIVERFILENAME}.so AliDriver-lib.go
chmod +x ${DRIVERFILENAME}.so
mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH
