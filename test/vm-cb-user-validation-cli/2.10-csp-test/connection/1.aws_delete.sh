#!/bin/bash
# Delete AWS connection info registered by ./1.aws.sh
cd "$(dirname "$0")" || exit 1
exec ./1.aws.sh delete
