#!/bin/bash
source $CBSPIDER_ROOT/setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=ncp-driver-v1.0

# To be built in the container
# cp $HOME/ncp/go.* ./
go mod download # cb-spider's go.mod and go.sum will be applied.

rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
go build -buildmode=plugin -o ${DRIVERFILENAME}.so NcpDriver-lib.go
chmod +x ${DRIVERFILENAME}.so
mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH
