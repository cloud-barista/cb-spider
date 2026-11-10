#!/bin/bash
# Register AWS connection info (driver, credential, region, connection config)
# - credential: $HOME/.cb-spider/.aws-credential (cf. ./.aws-credential.sample)
# - names: same as ~/connect-config scripts
# usage: ./1.aws.sh [register|delete]

cd "$(dirname "$0")" || exit 1
source ./common_conn.sh

PROVIDER="AWS"
DRIVER_NAME="aws-driver01"
DRIVER_LIB="aws-driver-v1.0.so"
CREDENTIAL_NAME="aws-credential01"
CREDENTIAL_FILE="$CRED_DIR/.aws-credential"
REQUIRED_VARS=("aws_access_key_id" "aws_secret_access_key")

credential_key_values() {
  echo '[
    {"Key": "aws_access_key_id", "Value": "'"$aws_access_key_id"'"},
    {"Key": "aws_secret_access_key", "Value": "'"$aws_secret_access_key"'"}
  ]'
}

# "RegionName:Region[:Zone]"
REGIONS=(
  "aws-ohio:us-east-2:us-east-2a"
  "aws-seoul:ap-northeast-2:ap-northeast-2a"
  "aws-sydney:ap-southeast-2:ap-southeast-2a"
  "aws-singapore:ap-southeast-1:ap-southeast-1a"
  "aws-paris:eu-west-3:eu-west-3a"
  "aws-saopaulo:sa-east-1:sa-east-1a"
  "aws-tokyo:ap-northeast-1:ap-northeast-1a"
)

# "ConfigName:RegionName"
CONFIGS=(
  "aws-config01:aws-sydney"
  "aws-seoul-config:aws-seoul"
  "aws-ohio-config:aws-ohio"
  "aws-singapore-config:aws-singapore"
  "aws-paris-config:aws-paris"
  "aws-saopaulo-config:aws-saopaulo"
  "aws-tokyo-config:aws-tokyo"
)

run_conn "$@"
