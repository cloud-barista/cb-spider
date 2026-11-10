#!/bin/bash
# Delete KT connection info registered by ./10.kt.sh
cd "$(dirname "$0")" || exit 1
exec ./10.kt.sh delete
