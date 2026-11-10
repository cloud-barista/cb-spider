#!/bin/bash
# Register ALIBABA connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.alibaba-credential (cf. ./.alibaba-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./4.alibaba.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="ALIBABA"
DRIVER_NAME="alibaba-driver01"
DRIVER_LIB="alibaba-driver-v1.0.so"
CREDENTIAL_NAME="alibaba-credential01"
CREDENTIAL_FILE="$CRED_DIR/.alibaba-credential"
REQUIRED_VARS=("alibaba_client_id" "alibaba_client_secret")

credential_key_values() {
  echo '[
    {"Key": "ClientId", "Value": "'"$alibaba_client_id"'"},
    {"Key": "ClientSecret", "Value": "'"$alibaba_client_secret"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "alibaba-beijing:cn-beijing:cn-beijing-f"
  "alibaba-ulanqab:cn-wulanchabu:cn-wulanchabu-1a"
  "alibaba-london:eu-west-1:eu-west-1a"
  "alibaba-tokyo:ap-northeast-1:ap-northeast-1a"
  "alibaba-singapore:ap-southeast-1:ap-southeast-1c"
  "alibaba-hongkong:cn-hongkong:cn-hongkong-c"
)

# "ConfigName:RegionName"
CONFIGS=(
  "alibaba-beijing-config:alibaba-beijing"
  "alibaba-ulanqab-config:alibaba-ulanqab"
  "alibaba-london-config:alibaba-london"
  "alibaba-tokyo-config:alibaba-tokyo"
  "alibaba-singapore-config:alibaba-singapore"
  "alibaba-hongkong-config:alibaba-hongkong"
)

run_conn "$@"
