#!/bin/bash
# Register NCP connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.ncp-credential (cf. ./.ncp-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./8.ncp.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="NCP"
DRIVER_NAME="ncp-driver01"
DRIVER_LIB="ncp-driver-v1.0.so"
CREDENTIAL_NAME="ncp-credential01"
CREDENTIAL_FILE="$CRED_DIR/.ncp-credential"
REQUIRED_VARS=("ncp_client_id" "ncp_client_secret")

credential_key_values() {
  echo '[
    {"Key": "ClientId", "Value": "'"$ncp_client_id"'"},
    {"Key": "ClientSecret", "Value": "'"$ncp_client_secret"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "ncp-korea1:KR:KR-1"
  "ncp-korea2:KR:KR-2"
  "ncp-singapore4:SGN:SGN-4"
  "ncp-singapore5:SGN:SGN-5"
  "ncp-japan4:JPN:JPN-4"
  "ncp-japan5:JPN:JPN-5"
)

# "ConfigName:RegionName"
CONFIGS=(
  "ncp-korea1-config:ncp-korea1"
  "ncp-korea2-config:ncp-korea2"
  "ncp-singapore4-config:ncp-singapore4"
  "ncp-singapore5-config:ncp-singapore5"
  "ncp-japan4-config:ncp-japan4"
  "ncp-japan5-config:ncp-japan5"
)

run_conn "$@"
