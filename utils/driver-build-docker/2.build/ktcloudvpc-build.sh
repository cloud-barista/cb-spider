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
# by CB-Spider Team, 2022.05.

# You have to run in driver build container.
echo "\$HOME" path is $HOME
echo "\$CBSPIDER_ROOT" path is $CBSPIDER_ROOT

echo "# cd" $CBSPIDER_ROOT
cd $CBSPIDER_ROOT

echo "# env GIT_TERMINAL_PROMPT=1 GOPRIVATE=github.com/cloud-barista go get -v github.com/cloud-barista/ktcloudvpc-sdk-for-drv@latest"
env GIT_TERMINAL_PROMPT=1 GOPRIVATE=github.com/cloud-barista go get -v github.com/cloud-barista/ktcloudvpc-sdk-for-drv@latest

cd $HOME

echo "# git clone https://github.com/cloud-barista/ktcloudvpc.git" $HOME"/ktcloudvpc;"
git clone https://github.com/cloud-barista/ktcloudvpc.git $HOME/ktcloudvpc;

ln -s $HOME/ktcloudvpc/ktcloudvpc $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers;
ln -s $HOME/ktcloudvpc/ktcloudvpc-plugin $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers;

echo "# cd "$CBSPIDER_ROOT"/cloud-control-manager/cloud-driver/drivers/ktcloudvpc-plugin;"
cd $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ktcloudvpc-plugin;

echo "# ./build_driver_lib.sh" 
./build_driver_lib.sh

rm $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ktcloudvpc;
rm $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ktcloudvpc-plugin;
