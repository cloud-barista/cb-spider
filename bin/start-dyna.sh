#!/bin/bash

# start CB-Spider Server with Driver Plugin Mode.
#
# The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
# The CB-Spider Mission is to connect all the clouds with a single interface.
#
#      * Cloud-Barista: https://github.com/cloud-barista
#
# by CB-Spider Team, 2021.04.

#### Load CB-Spider Environment Variables
source ../setup.env

### Set the library type of Cloud Driver pkg.
# ON is a shared library driver type.
# OFF is a static package type.
export PLUGIN_SW=ON

./stop.sh &> /dev/null

echo -e '\n'
echo -e '\t[CB-Spider] Driver Plugin Mode: Dynamic Plugin Mode'
echo -e '\n'

$CBSPIDER_ROOT/bin/cb-spider-dyna &
echo $! > spider.pid
