#!/bin/bash
# Register IBM connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.ibm-credential (cf. ./.ibm-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./6.ibm.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="IBM"
DRIVER_NAME="ibm-driver01"
DRIVER_LIB="ibm-driver-v1.0.so"
CREDENTIAL_NAME="ibm-credential01"
CREDENTIAL_FILE="$CRED_DIR/.ibm-credential"
REQUIRED_VARS=("ibm_api_key")
S3_VARS=("ibm_access_key_id" "ibm_secret_access_key")   # keys for Object Storage(S3)

credential_key_values() {
  echo '[
    {"Key": "ApiKey", "Value": "'"$ibm_api_key"'"},
    {"Key": "access_key_id", "Value": "'"$ibm_access_key_id"'"},
    {"Key": "secret_access_key", "Value": "'"$ibm_secret_access_key"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "ibm-us-south-1:us-south:us-south-1"
  "ibm-us-south-2:us-south:us-south-2"
  "ibm-us-south-3:us-south:us-south-3"
  "ibm-br-sao-1:br-sao:br-sao-1"
  "ibm-br-sao-2:br-sao:br-sao-2"
  "ibm-br-sao-3:br-sao:br-sao-3"
  "ibm-ca-tor-1:ca-tor:ca-tor-1"
  "ibm-ca-tor-2:ca-tor:ca-tor-2"
  "ibm-ca-tor-3:ca-tor:ca-tor-3"
  "ibm-us-east-1:us-east:us-east-1"
  "ibm-us-east-2:us-east:us-east-2"
  "ibm-us-east-3:us-east:us-east-3"
  "ibm-eu-de-1:eu-de:eu-de-1"
  "ibm-eu-de-2:eu-de:eu-de-2"
  "ibm-eu-de-3:eu-de:eu-de-3"
  "ibm-eu-gb-1:eu-gb:eu-gb-1"
  "ibm-eu-gb-2:eu-gb:eu-gb-2"
  "ibm-eu-gb-3:eu-gb:eu-gb-3"
  "ibm-jp-osa-1:jp-osa:jp-osa-1"
  "ibm-jp-osa-2:jp-osa:jp-osa-2"
  "ibm-jp-osa-3:jp-osa:jp-osa-3"
  "ibm-au-syd-1:au-syd:au-syd-1"
  "ibm-au-syd-2:au-syd:au-syd-2"
  "ibm-au-syd-3:au-syd:au-syd-3"
  "ibm-jp-tok-1:jp-tok:jp-tok-1"
  "ibm-jp-tok-2:jp-tok:jp-tok-2"
  "ibm-jp-tok-3:jp-tok:jp-tok-3"
)

# "ConfigName:RegionName"
CONFIGS=(
  "ibm-us-south-1-config:ibm-us-south-1"
  "ibm-us-south-2-config:ibm-us-south-2"
  "ibm-us-south-3-config:ibm-us-south-3"
  "ibm-br-sao-1-config:ibm-br-sao-1"
  "ibm-br-sao-2-config:ibm-br-sao-2"
  "ibm-br-sao-3-config:ibm-br-sao-3"
  "ibm-ca-tor-1-config:ibm-ca-tor-1"
  "ibm-ca-tor-2-config:ibm-ca-tor-2"
  "ibm-ca-tor-3-config:ibm-ca-tor-3"
  "ibm-us-east-1-config:ibm-us-east-1"
  "ibm-us-east-2-config:ibm-us-east-2"
  "ibm-us-east-3-config:ibm-us-east-3"
  "ibm-eu-de-1-config:ibm-eu-de-1"
  "ibm-eu-de-2-config:ibm-eu-de-2"
  "ibm-eu-de-3-config:ibm-eu-de-3"
  "ibm-eu-gb-1-config:ibm-eu-gb-1"
  "ibm-eu-gb-2-config:ibm-eu-gb-2"
  "ibm-eu-gb-3-config:ibm-eu-gb-3"
  "ibm-jp-osa-1-config:ibm-jp-osa-1"
  "ibm-jp-osa-2-config:ibm-jp-osa-2"
  "ibm-jp-osa-3-config:ibm-jp-osa-3"
  "ibm-au-syd-1-config:ibm-au-syd-1"
  "ibm-au-syd-2-config:ibm-au-syd-2"
  "ibm-au-syd-3-config:ibm-au-syd-3"
  "ibm-jp-tok-1-config:ibm-jp-tok-1"
  "ibm-jp-tok-2-config:ibm-jp-tok-2"
  "ibm-jp-tok-3-config:ibm-jp-tok-3"
)

run_conn "$@"
