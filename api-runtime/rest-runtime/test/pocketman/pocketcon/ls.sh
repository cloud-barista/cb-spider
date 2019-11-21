#!/bin/bash
source setup.env

#SHOOTERS=( aws-shooter-name azure-shooter gcp-shooter openstack-shooter-name cloudit-shooter cloudtwin-shooter )
SHOOTERS=( aws-shooter-name azure-shooter gcp-shooter openstack-shooter-name cloudit-shooter )


num=0
for SHOOTER in "${SHOOTERS[@]}"
do
        #echo  ============ build ${DRIVER} ... ============
        #DRIVER_BUILD_SHELL=$DRIVER_PATH/$DRIVER/build_driver_lib.sh
        #/bin/bash -c 'cd '$DRIVER_PATH/$DRIVER';'$DRIVER_BUILD_SHELL'' &

	ls $TEST_PATH/$SHOOTER/vnetwork
        num=`expr $num + 1`
done

