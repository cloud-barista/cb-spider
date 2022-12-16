#!/bin/bash

# Build Scripts to build drivers with plugin mode.
# The driver repos can be other repos.
# cf) https://github.com/cloud-barista/cb-spider/issues/343
#
# The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
# The CB-Spider Mission is to connect all the clouds with a single interface.
#
#      * Cloud-Barista: https://github.com/cloud-barista
#
# by CB-Spider Team, 2022.01.

# You have to run in driver build container.
echo "\$HOME" path is $HOME
echo "\$CBSPIDER_ROOT" path is $CBSPIDER_ROOT

echo "# cd" $CBSPIDER_ROOT
cd $CBSPIDER_ROOT

echo "# go get -v github.com/NaverCloudPlatform/ncloud-sdk-go-v2@v1.5.3"
go get -v github.com/NaverCloudPlatform/ncloud-sdk-go-v2@v1.5.3

cd $HOME

echo "# git clone https://github.com/cloud-barista/ncpvpc.git" $HOME"/ncpvpc;"
git clone https://github.com/cloud-barista/ncpvpc.git $HOME/ncpvpc;

ln -s $HOME/ncpvpc/ncpvpc $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers;
ln -s $HOME/ncpvpc/ncpvpc-plugin $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers;

echo "# cd "$CBSPIDER_ROOT"/cloud-control-manager/cloud-driver/drivers/ncpvpc-plugin;"
cd $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ncpvpc-plugin;

echo "# ./build_driver_lib.sh" 
./build_driver_lib.sh

rm $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ncpvpc;
rm $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ncpvpc-plugin;
