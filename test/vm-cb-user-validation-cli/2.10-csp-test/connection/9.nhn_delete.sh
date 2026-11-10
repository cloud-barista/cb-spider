#!/bin/bash
# Delete NHN connection info registered by ./9.nhn.sh
cd "$(dirname "$0")" || exit 1
exec ./9.nhn.sh delete
