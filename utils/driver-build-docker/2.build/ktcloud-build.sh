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
# by CB-Spider Team, 2021.05.

# You have to run in driver build container.
echo "\$HOME" path is $HOME
echo "\$CBSPIDER_ROOT" path is $CBSPIDER_ROOT

echo "# cd" $CBSPIDER_ROOT
cd $CBSPIDER_ROOT

env GIT_TERMINAL_PROMPT=1 GOPRIVATE=github.com/cloud-barista

echo "# go get -v github.com/cloud-barista/ktcloud-sdk-go@latest"
go get -v github.com/cloud-barista/ktcloud-sdk-go@latest

cd $HOME

echo "# git clone https://github.com/cloud-barista/ktcloud.git" $HOME"/ktcloud;"
git clone https://github.com/cloud-barista/ktcloud.git $HOME/ktcloud;

ln -s $HOME/ktcloud/ktcloud $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers;
ln -s $HOME/ktcloud/ktcloud-plugin $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers;

echo "# cd "$CBSPIDER_ROOT"/cloud-control-manager/cloud-driver/drivers/ktcloud-plugin;"
cd $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ktcloud-plugin;

echo "# ./build_driver_lib.sh" 
./build_driver_lib.sh

rm $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ktcloud;
rm $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ktcloud-plugin;
