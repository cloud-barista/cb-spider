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
# by CB-Spider Team, 2021.04.


# You have to run in driver build container.
echo "\$HOME" path is $HOME
echo "\$CBSPIDER_ROOT" path is $CBSPIDER_ROOT

git clone https://github.com/cloud-barista/cloud-twin.git $HOME/cloud-twin;

ln -s $HOME/cloud-twin/cloudtwin $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers;
ln -s $HOME/cloud-twin/cloudtwin-plugin $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers;

cd $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/cloudtwin-plugin;
./build_driver_lib.sh

rm $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/cloudtwin;
rm $CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/cloudtwin-plugin;

