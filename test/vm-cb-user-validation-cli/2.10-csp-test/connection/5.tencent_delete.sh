#!/bin/bash
# Delete TENCENT connection info registered by ./5.tencent.sh
cd "$(dirname "$0")" || exit 1
exec ./5.tencent.sh delete
