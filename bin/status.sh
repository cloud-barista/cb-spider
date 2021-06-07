#!/bin/bash

#  support endpoint information of current CB-Spider Server 
#
# The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
# The CB-Spider Mission is to connect all the clouds with a single interface.
#
#      * Cloud-Barista: https://github.com/cloud-barista
#
# by CB-Spider Team, 2021.05.

#### Load CB-Spider Environment Variables
source ../setup.env

# slow, but no curl
#$CBSPIDER_ROOT/bin/cb-spider info 2> /dev/null
# fast, but need curl
curl http://localhost:1024/spider/endpointinfo
echo
