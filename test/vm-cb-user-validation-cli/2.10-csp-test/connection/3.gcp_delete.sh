#!/bin/bash
# Delete GCP connection info registered by ./3.gcp.sh
cd "$(dirname "$0")" || exit 1
exec ./3.gcp.sh delete
