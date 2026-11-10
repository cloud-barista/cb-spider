#!/bin/bash
# Delete NCP connection info registered by ./8.ncp.sh
cd "$(dirname "$0")" || exit 1
exec ./8.ncp.sh delete
