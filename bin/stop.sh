#!/bin/bash

# stop CB-Spider Server
#
# The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
# The CB-Spider Mission is to connect all the clouds with a single interface.
#
#      * Cloud-Barista: https://github.com/cloud-barista
#
# by CB-Spider Team, 2021.04.

kill -9 `cat spider.pid` &> /dev/null
rm -rf spider.pid
