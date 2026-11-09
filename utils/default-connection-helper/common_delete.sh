#!/bin/bash
SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


# Check if RESTSERVER is set, otherwise default to localhost
if [[ -z "$RESTSERVER" ]]; then
  echo "RESTSERVER is not set. Defaulting to 'localhost'."
  RESTSERVER="localhost"
fi

# Helper: DELETE a Spider resource and print the result
# usage: spider_delete <resource-path> <label>
spider_delete() {
  local RESOURCE_PATH=$1
  local LABEL=$2
  local RESP HTTP_CODE BODY

  # Spider returns 200 even when deleting a non-existent resource, so check existence first
  HTTP_CODE=$(curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -s -o /dev/null -w '%{http_code}' -X GET \
    "http://$RESTSERVER:1024/spider/$RESOURCE_PATH" -H 'accept: application/json')
  if [[ "$HTTP_CODE" != 2* ]]; then
    echo "[SKIP] $LABEL (not found)"
    return 0
  fi

  RESP=$(curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -s -w '\n%{http_code}' -X DELETE \
    "http://$RESTSERVER:1024/spider/$RESOURCE_PATH" -H 'Content-Type: application/json')
  HTTP_CODE=$(echo "$RESP" | tail -n1)
  BODY=$(echo "$RESP" | sed '$d')

  if [[ "$HTTP_CODE" == 2* ]]; then
    echo "Deleted $LABEL"
  else
    echo "[SKIP/FAIL] $LABEL (HTTP $HTTP_CODE) $(echo "$BODY" | jq -r '.message // empty' 2>/dev/null)"
  fi
}

# Function to delete connection configs and regions
# - Targets are taken from what is actually registered in Spider (not from the CSP),
#   so cleanup works even if the driver/credential is missing or invalid.
# - Only names created by *_register.sh ("$PREFIX.<region>.<zone>") are deleted.
delete_connection_configs_and_regions() {
  local DRIVER_NAME=$1
  local CREDENTIAL_NAME=$2
  local PREFIX=$3

  echo "####################################################################"
  echo "## Cloud Connection Config Deletion"
  echo "####################################################################"

  CONFIG_JSON=$(curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -s -X GET \
    "http://$RESTSERVER:1024/spider/connectionconfig" -H 'accept: application/json')

  if ! echo "$CONFIG_JSON" | jq -e '.connectionconfig' > /dev/null 2>&1; then
    echo "[ERROR] Failed to list connection configs."
    echo "$CONFIG_JSON" | jq . 2>/dev/null || echo "$CONFIG_JSON"
  else
    CONFIG_NAMES=$(echo "$CONFIG_JSON" | jq -r --arg d "$DRIVER_NAME" --arg p "$PREFIX." \
      '.connectionconfig // [] | .[] | select(.DriverName == $d and (.ConfigName | startswith($p))) | .ConfigName')
    if [[ -z "$CONFIG_NAMES" ]]; then
      echo "No connection configs found for driver '$DRIVER_NAME' (prefix '$PREFIX.')."
    else
      for CONFIG_NAME in $CONFIG_NAMES; do
        spider_delete "connectionconfig/$CONFIG_NAME" "connection config: $CONFIG_NAME"
      done
    fi
  fi

  echo "####################################################################"
  echo "## Cloud Region Deletion"
  echo "####################################################################"

  REGION_JSON=$(curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -s -X GET \
    "http://$RESTSERVER:1024/spider/region" -H 'accept: application/json')

  if ! echo "$REGION_JSON" | jq -e '.region' > /dev/null 2>&1; then
    echo "[ERROR] Failed to list regions."
    echo "$REGION_JSON" | jq . 2>/dev/null || echo "$REGION_JSON"
  else
    REGION_NAMES=$(echo "$REGION_JSON" | jq -r --arg p "$PREFIX." \
      '.region // [] | .[] | select(.RegionName | startswith($p)) | .RegionName')
    if [[ -z "$REGION_NAMES" ]]; then
      echo "No regions found with prefix '$PREFIX.'."
    else
      for REGION_NAME in $REGION_NAMES; do
        spider_delete "region/$REGION_NAME" "region: $REGION_NAME"
      done
    fi
  fi
}

# Function to delete credential
delete_credential() {
  local CREDENTIAL_NAME=$1

  echo "####################################################################"
  echo "## Cloud Credential Deletion"
  echo "####################################################################"

  spider_delete "credential/$CREDENTIAL_NAME" "credential: $CREDENTIAL_NAME"
}

# Function to delete driver
delete_driver() {
  local DRIVER_NAME=$1

  echo "####################################################################"
  echo "## Cloud Driver Deletion"
  echo "####################################################################"

  spider_delete "driver/$DRIVER_NAME" "driver: $DRIVER_NAME"
}

# Function to display completion message
show_completion() {
  echo "####################################################################"
  echo "## All Deletion Tasks Completed"
  echo "####################################################################"
  echo -e "\n"
}
