#!/bin/bash
# Register NHN connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.nhn-credential (cf. ./.nhn-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./9.nhn.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="NHN"
DRIVER_NAME="nhn-driver01"
DRIVER_LIB="nhn-driver-v1.0.so"
CREDENTIAL_NAME="nhn-credential01"
CREDENTIAL_FILE="$CRED_DIR/.nhn-credential"
REQUIRED_VARS=("nhn_identity_endpoint" "nhn_username" "nhn_password" "nhn_domain_name" "nhn_tenant_id")
S3_VARS=("nhn_access_key" "nhn_secret_key")   # keys for Object Storage(S3)

credential_key_values() {
  # NHN driver requires the versioned identity endpoint (ex: https://api-identity-infrastructure.nhncloudservice.com/v2.0)
  # => append "/v2.0" when the endpoint has no version path
  nhn_identity_endpoint="${nhn_identity_endpoint%/}"
  [[ "$nhn_identity_endpoint" =~ /v[0-9]+(\.[0-9]+)?$ ]] || nhn_identity_endpoint="${nhn_identity_endpoint}/v2.0"
  echo '[
    {"Key": "IdentityEndpoint", "Value": "'"$nhn_identity_endpoint"'"},
    {"Key": "Username", "Value": "'"$nhn_username"'"},
    {"Key": "Password", "Value": "'"$nhn_password"'"},
    {"Key": "DomainName", "Value": "'"$nhn_domain_name"'"},
    {"Key": "TenantId", "Value": "'"$nhn_tenant_id"'"},
    {"Key": "Access Key", "Value": "'"$nhn_access_key"'"},
    {"Key": "Secret Key", "Value": "'"$nhn_secret_key"'"},
    {"Key": "User Access Key", "Value": "'"$nhn_rds_user_access_key"'"},
    {"Key": "Secret Access Key", "Value": "'"$nhn_rds_secret_access_key"'"},
    {"Key": "mysqlAppKey", "Value": "'"$nhn_rds_mysql_app_key"'"},
    {"Key": "mariadbAppKey", "Value": "'"$nhn_rds_mariadb_app_key"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "nhn-korea-pangyo1:KR1:kr-pub-a"
  "nhn-korea-pangyo2:KR1:kr-pub-b"
  "nhn-korea-pyeongchon1:KR2:kr2-pub-a"
  "nhn-korea-pyeongchon2:KR2:kr2-pub-b"
  "nhn-japan-tokyo1:JP1:jp-pub-a"
  "nhn-japan-tokyo2:JP1:jp-pub-b"
)

# "ConfigName:RegionName"
CONFIGS=(
  "nhn-korea-pangyo1-config:nhn-korea-pangyo1"
  "nhn-korea-pangyo2-config:nhn-korea-pangyo2"
  "nhn-korea-pyeongchon1-config:nhn-korea-pyeongchon1"
  "nhn-korea-pyeongchon2-config:nhn-korea-pyeongchon2"
  "nhn-japan-tokyo1-config:nhn-japan-tokyo1"
  "nhn-japan-tokyo2-config:nhn-japan-tokyo2"
)

run_conn "$@"
