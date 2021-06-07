#!/bin/bash
source setup.env

#DRIVERS=( aws-plugin azure-plugin openstack-plugin gcp-plugin alibaba-plugin cloudit-plugin docker-plugin ncp-plugin ncpvpc-plugin)
DRIVERS=( aws-plugin azure-plugin gcp-plugin alibaba-plugin openstack-plugin cloudit-plugin docker-plugin mock-plugin tencent-plugin )

DRIVER_PATH=$CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers
DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs

function build() {
        num=0
        for DRIVER in "${DRIVERS[@]}"
        do
                echo  ============ build ${DRIVER} ... ============
                DRIVER_BUILD_SHELL=$DRIVER_PATH/$DRIVER/build_driver_lib.sh
                /bin/bash -c 'cd '$DRIVER_PATH/$DRIVER';'$DRIVER_BUILD_SHELL'' || return 1

                num=`expr $num + 1`
        done
        }

build
