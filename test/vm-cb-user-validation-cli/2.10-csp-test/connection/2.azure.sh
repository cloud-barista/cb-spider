#!/bin/bash
# Register AZURE connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.azure-credential (cf. ./.azure-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./2.azure.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="AZURE"
DRIVER_NAME="azure-driver01"
DRIVER_LIB="azure-driver-v1.0.so"
CREDENTIAL_NAME="azure-credential01"
CREDENTIAL_FILE="$CRED_DIR/.azure-credential"
REQUIRED_VARS=("azure_client_id" "azure_client_secret" "azure_tenant_id" "azure_subscription_id")
S3_VARS=("azure_storage_account_name" "azure_storage_account_key")   # keys for Object Storage(S3)

credential_key_values() {
  echo '[
    {"Key": "ClientId", "Value": "'"$azure_client_id"'"},
    {"Key": "ClientSecret", "Value": "'"$azure_client_secret"'"},
    {"Key": "TenantId", "Value": "'"$azure_tenant_id"'"},
    {"Key": "SubscriptionId", "Value": "'"$azure_subscription_id"'"},
    {"Key": "StorageAccountName", "Value": "'"$azure_storage_account_name"'"},
    {"Key": "StorageAccountKey", "Value": "'"$azure_storage_account_key"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "azure-northeu:northeurope:1"
  "azure-eastus:eastus:1"
  "azure-japanwest:japanwest:1"
  "azure-koreacentral:koreacentral:1"
)

# "ConfigName:RegionName"
CONFIGS=(
  "azure-northeu-config:azure-northeu"
  "azure-eastus-config:azure-eastus"
  "azure-japanwest-config:azure-japanwest"
  "azure-koreacentral-config:azure-koreacentral"
)

run_conn "$@"
