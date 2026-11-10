#!/bin/bash
# Delete OPENSTACK connection info registered by ./7.openstack.sh
cd "$(dirname "$0")" || exit 1
exec ./7.openstack.sh delete
