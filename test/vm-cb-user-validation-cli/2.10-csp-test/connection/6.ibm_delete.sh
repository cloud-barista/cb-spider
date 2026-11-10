#!/bin/bash
# Delete IBM connection info registered by ./6.ibm.sh
cd "$(dirname "$0")" || exit 1
exec ./6.ibm.sh delete
