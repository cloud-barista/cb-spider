#!/bin/bash
# Delete AZURE connection info registered by ./2.azure.sh
cd "$(dirname "$0")" || exit 1
exec ./2.azure.sh delete
