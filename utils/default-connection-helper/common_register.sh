#!/bin/bash

# Check if RESTSERVER is set, otherwise default to localhost
if [[ -z "$RESTSERVER" ]]; then
  echo "RESTSERVER is not set. Defaulting to 'localhost'."
  RESTSERVER="localhost"
fi

# Function to register cloud driver
register_driver() {
  local DRIVER_NAME=$1
  local PROVIDER_NAME=$2
  local DRIVER_LIB_FILE=$3

  echo "####################################################################"
  echo "## Cloud Driver Registration"
  echo "####################################################################"

  curl -s -X POST http://$RESTSERVER:1024/spider/driver \
    -H 'Content-Type: application/json' \
    -d '{
    "DriverName":"'"$DRIVER_NAME"'",
    "ProviderName":"'"$PROVIDER_NAME"'",
    "DriverLibFileName":"'"$DRIVER_LIB_FILE"'"
  }' > /dev/null 2>&1
}

# Function to register cloud credential
register_credential() {
  local CREDENTIAL_NAME=$1
  local PROVIDER_NAME=$2
  local KEY_VALUE_LIST=$3

  echo "####################################################################"
  echo "## Cloud Credential Registration"
  echo "####################################################################"

  curl -s -X POST http://$RESTSERVER:1024/spider/credential \
    -H 'Content-Type: application/json' \
    -d '{
    "CredentialName":"'"$CREDENTIAL_NAME"'",
    "ProviderName":"'"$PROVIDER_NAME"'",
    "KeyValueInfoList": '"$KEY_VALUE_LIST"'
  }' > /dev/null 2>&1
}

# Function to register regions and connection configs
register_regions_and_configs() {
  local DRIVER_NAME=$1
  local CREDENTIAL_NAME=$2
  local PROVIDER_NAME=$3
  local PREFIX=$4

  echo "####################################################################"
  echo "## Cloud Region & Connection Config Registration (Parallel)"
  echo "####################################################################"

  REGIONZONE_JSON=$(curl -s -X GET \
    "http://$RESTSERVER:1024/spider/preconfig/regionzone?CredentialName=$CREDENTIAL_NAME&DriverName=$DRIVER_NAME" \
    -H 'accept: application/json')

  if ! echo "$REGIONZONE_JSON" | jq -e '.regionzone | length > 0' > /dev/null; then
    echo "[ERROR] Failed to fetch valid regionzone list. Response:"
    echo "$REGIONZONE_JSON" | jq .
    exit 1
  fi

  TMP_DONE_LOG=$(mktemp)

  echo "$REGIONZONE_JSON" | jq -r '
  .regionzone[] | .Name as $region |
  select(.ZoneList != null) |
  .ZoneList[] | "\($region):\(.Name)"' | while IFS=":" read -r REGION ZONE; do
  (
    REGION_ID="$PREFIX.${REGION}.${ZONE}"
    echo "[START] $REGION_ID"

    curl -s -X POST http://$RESTSERVER:1024/spider/region \
      -H 'Content-Type: application/json' \
      -d '{
        "RegionName": "'"$REGION_ID"'",
        "ProviderName": "'"$PROVIDER_NAME"'",
        "KeyValueInfoList": [
          {"Key": "Region", "Value": "'"$REGION"'"},
          {"Key": "Zone", "Value": "'"$ZONE"'"}
        ]
      }' > /dev/null 2>&1

    curl -s -X POST http://$RESTSERVER:1024/spider/connectionconfig \
      -H 'Content-Type: application/json' \
      -d '{
        "ConfigName": "'"$REGION_ID"'",
        "ProviderName": "'"$PROVIDER_NAME"'",
        "DriverName": "'"$DRIVER_NAME"'",
        "CredentialName": "'"$CREDENTIAL_NAME"'",
        "RegionName": "'"$REGION_ID"'"
      }' > /dev/null 2>&1

    echo "[DONE]  $REGION_ID" >> "$TMP_DONE_LOG"
  ) &
  done

  wait
  sort "$TMP_DONE_LOG"
  rm -f "$TMP_DONE_LOG"

  echo "####################################################################"
  echo "## All region and connection registrations completed"
  echo "####################################################################"
  echo -e "\n"
}

# Function to check if a credential file exists and load it
check_credential_file() {
  local CREDENTIAL_FILE=$1
  local ERROR_MSG=$2

  if [[ -f $CREDENTIAL_FILE ]]; then
    source $CREDENTIAL_FILE
  else
    echo "Error: Credential file not found at $CREDENTIAL_FILE"
    exit 1
  fi
}

# Function to check if required variables are set
check_required_vars() {
  local MISSING=0

  for VAR in "$@"; do
    if [[ -z "${!VAR}" ]]; then
      echo "Error: Missing required variable $VAR"
      MISSING=1
    fi
  done

  if [[ $MISSING -eq 1 ]]; then
    echo "Error: Missing one or more required credential variables in credential file."
    exit 1
  fi
}
