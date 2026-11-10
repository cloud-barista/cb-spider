#!/bin/bash
# Register KT connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.kt-credential (cf. ./.kt-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./10.kt.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="KT"
DRIVER_NAME="kt-driver01"
DRIVER_LIB="kt-driver-v1.0.so"
CREDENTIAL_NAME="kt-credential01"
CREDENTIAL_FILE="$CRED_DIR/.kt-credential"
REQUIRED_VARS=("kt_identity_endpoint" "kt_username" "kt_password" "kt_domain_name" "kt_project_id")
S3_VARS=("kt_access_key" "kt_secret_key")   # keys for Object Storage(S3)

credential_key_values() {
  echo '[
    {"Key": "IdentityEndpoint", "Value": "'"$kt_identity_endpoint"'"},
    {"Key": "Username", "Value": "'"$kt_username"'"},
    {"Key": "Password", "Value": "'"$kt_password"'"},
    {"Key": "DomainName", "Value": "'"$kt_domain_name"'"},
    {"Key": "ProjectID", "Value": "'"$kt_project_id"'"},
    {"Key": "Access Key", "Value": "'"$kt_access_key"'"},
    {"Key": "Secret Key", "Value": "'"$kt_secret_key"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "kt-DX-M1-zone:KR1:DX-M1"
)

# "ConfigName:RegionName"
CONFIGS=(
  "kt-mokdong1-config:kt-DX-M1-zone"
)

run_conn "$@"
