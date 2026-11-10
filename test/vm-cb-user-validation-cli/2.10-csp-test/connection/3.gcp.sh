#!/bin/bash
# Register GCP connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.gcp-credential (cf. ./.gcp-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./3.gcp.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="GCP"
DRIVER_NAME="gcp-driver01"
DRIVER_LIB="gcp-driver-v1.0.so"
CREDENTIAL_NAME="gcp-credential01"
CREDENTIAL_FILE="$CRED_DIR/.gcp-credential"
REQUIRED_VARS=("gcp_private_key" "gcp_project_id" "gcp_client_email")
S3_VARS=("gcp_access_key" "gcp_secret")   # keys for Object Storage(S3)

credential_key_values() {
  echo '[
    {"Key": "PrivateKey", "Value": "'"$gcp_private_key"'"},
    {"Key": "ProjectID", "Value": "'"$gcp_project_id"'"},
    {"Key": "ClientEmail", "Value": "'"$gcp_client_email"'"},
    {"Key": "Access Key", "Value": "'"$gcp_access_key"'"},
    {"Key": "Secret", "Value": "'"$gcp_secret"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "gcp-iowa:us-central1:us-central1-a"
  "gcp-oregon:us-west1:us-west1-a"
  "gcp-london:europe-west2:europe-west2-a"
  "gcp-tokyo:asia-northeast1:asia-northeast1-a"
  "gcp-seoul:asia-northeast3:asia-northeast3-a"
)

# "ConfigName:RegionName"
CONFIGS=(
  "gcp-iowa-config:gcp-iowa"
  "gcp-oregon-config:gcp-oregon"
  "gcp-london-config:gcp-london"
  "gcp-tokyo-config:gcp-tokyo"
  "gcp-seoul-config:gcp-seoul"
)

run_conn "$@"
