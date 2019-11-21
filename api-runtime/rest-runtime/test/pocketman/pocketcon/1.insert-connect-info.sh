#!/bin/bash
source setup.env

SHOOTERS=( aws-shooter-name azure-shooter gcp-shooter openstack-shooter-name cloudit-shooter cloudtwin-shooter )
WORK_PATH=$TEST_PATH/$SHOOTER
CMD=cim-insert-test.sh.value

num=0
for SHOOTER in "${SHOOTERS[@]}"
do
        /bin/bash -c 'cd '$WORK_PATH';'./$CMD'' 

        num=`expr $num + 1`
done


SHOOTERS=( aws-shooter-name azure-shooter gcp-shooter )
CMD=insert-all-regions.sh
num=0
for SHOOTER in "${SHOOTERS[@]}"
do

        DRIVER_BUILD_SHELL=$DRIVER_PATH/$DRIVER/build_driver_lib.sh
        /bin/bash -c 'cd '$TEST_PATH/$SHOOTER';'./$CMD''

        num=`expr $num + 1`
done

