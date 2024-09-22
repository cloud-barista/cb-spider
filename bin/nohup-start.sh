#!/bin/bash

# start CB-Spider Server.
#
# The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
# The CB-Spider Mission is to connect all the clouds with a single interface.
#
#      * Cloud-Barista: https://github.com/cloud-barista
#
# by CB-Spider Team, 2021.04.

#### Load CB-Spider Environment Variables
SCRIPT_DIR=`dirname ${BASH_SOURCE[0]-$0}`
BIN_DIR=`cd $SCRIPT_DIR && pwd`
source ${BIN_DIR}/../setup.env

### Set the library type of Cloud Driver pkg.
# ON is a shared library type.
# OFF is a static package type.
export PLUGIN_SW=OFF

# If arguments are provided, pass them to cb-spider
if [ "$#" -gt 0 ]; then
    $BIN_DIR/cb-spider "$@"
else
  # If no arguments are provided, start the CB-Spider server
  # Stop any running instance of cb-spider
  ${BIN_DIR}/stop.sh &> /dev/null


  echo -e '\n'
  echo -e '\t[CB-Spider] Driver Plugin Mode: Static Builtin Mode'
  echo -e '\n'

  nohup $BIN_DIR/cb-spider > spider-nohup.out 2>&1  &
  echo $! > $BIN_DIR/spider.pid
fi