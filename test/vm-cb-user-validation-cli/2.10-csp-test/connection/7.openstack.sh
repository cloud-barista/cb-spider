#!/bin/bash
# Register OPENSTACK connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.openstack-credential (cf. ./.openstack-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./7.openstack.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="OPENSTACK"
DRIVER_NAME="openstack-driver01"
DRIVER_LIB="openstack-driver-v1.0.so"
CREDENTIAL_NAME="openstack-credential01"
CREDENTIAL_FILE="$CRED_DIR/.openstack-credential"
REQUIRED_VARS=("openstack_identity_endpoint" "openstack_username" "openstack_password" "openstack_domain_name" "openstack_project_id")
S3_VARS=("openstack_access" "openstack_secret")   # keys for Object Storage(S3)

credential_key_values() {
  echo '[
    {"Key": "IdentityEndpoint", "Value": "'"$openstack_identity_endpoint"'"},
    {"Key": "Username", "Value": "'"$openstack_username"'"},
    {"Key": "Password", "Value": "'"$openstack_password"'"},
    {"Key": "DomainName", "Value": "'"$openstack_domain_name"'"},
    {"Key": "ProjectID", "Value": "'"$openstack_project_id"'"},
    {"Key": "access", "Value": "'"$openstack_access"'"},
    {"Key": "secret", "Value": "'"$openstack_secret"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "openstack-region01:RegionOne:nova"
)

# "ConfigName:RegionName"
CONFIGS=(
  "openstack-config01:openstack-region01"
)

run_conn "$@"
