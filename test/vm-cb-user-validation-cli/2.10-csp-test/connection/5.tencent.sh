#!/bin/bash
# Register TENCENT connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.tencent-credential (cf. ./.tencent-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./5.tencent.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="TENCENT"
DRIVER_NAME="tencent-driver01"
DRIVER_LIB="tencent-driver-v1.0.so"
CREDENTIAL_NAME="tencent-credential01"
CREDENTIAL_FILE="$CRED_DIR/.tencent-credential"
REQUIRED_VARS=("tencent_secret_id" "tencent_secret_key")

credential_key_values() {
  echo '[
    {"Key": "SecretId", "Value": "'"$tencent_secret_id"'"},
    {"Key": "SecretKey", "Value": "'"$tencent_secret_key"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "tencent-guangzhou-3:ap-guangzhou:ap-guangzhou-6"
  "tencent-beijing-3:ap-beijing:ap-beijing-3"
  "tencent-beijing-6:ap-beijing:ap-beijing-6"
  "tencent-seoul-1:ap-seoul:ap-seoul-1"
  "tencent-tokyo-1:ap-tokyo:ap-tokyo-1"
  "tencent-frankfurt-1:eu-frankfurt:eu-frankfurt-1"
)

# "ConfigName:RegionName"
CONFIGS=(
  "tencent-guangzhou3-config:tencent-guangzhou-3"
  "tencent-beijing3-config:tencent-beijing-3"
  "tencent-beijing6-config:tencent-beijing-6"
  "tencent-seoul1-config:tencent-seoul-1"
  "tencent-tokyo-config:tencent-tokyo-1"
  "tencent-frankfurt1-config:tencent-frankfurt-1"
)

run_conn "$@"
