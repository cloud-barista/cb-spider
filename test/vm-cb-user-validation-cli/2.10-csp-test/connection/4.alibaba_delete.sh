#!/bin/bash
# Delete ALIBABA connection info registered by ./4.alibaba.sh
cd "$(dirname "$0")" || exit 1
exec ./4.alibaba.sh delete
