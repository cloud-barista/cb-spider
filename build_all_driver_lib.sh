#!/bin/bash
source setup.env

#DRIVERS=( aws azure cloudit gcp openstack cloudtwin )
DRIVERS=( aws azure cloudit gcp openstack )

DRIVER_PATH=$CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers
DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs

num=0
for DRIVER in "${DRIVERS[@]}"
do
        echo  ============ build ${DRIVER} ... ============
        DRIVER_BUILD_SHELL=$DRIVER_PATH/$DRIVER/build_driver_lib.sh
        /bin/bash -c 'cd '$DRIVER_PATH/$DRIVER';'$DRIVER_BUILD_SHELL'' &

        num=`expr $num + 1`
done
