#!/bin/bash

# Check if RESTSERVER is set, otherwise default to localhost
if [[ -z "$RESTSERVER" ]]; then
  echo "RESTSERVER is not set. Defaulting to 'localhost'."
  RESTSERVER="localhost"
fi

# Function to delete connection configs and regions
delete_connection_configs_and_regions() {
  local DRIVER_NAME=$1
  local CREDENTIAL_NAME=$2
  local PREFIX=$3

  echo "####################################################################"
  echo "## Cloud Connection Config Deletion"
  echo "####################################################################"

  # Fetch region-zone list used to build config/region names
  REGIONZONE_JSON=$(curl -s -X GET \
    "http://$RESTSERVER:1024/spider/preconfig/regionzone?CredentialName=$CREDENTIAL_NAME&DriverName=$DRIVER_NAME" \
    -H 'accept: application/json')

  # Check if .regionzone is present and is an array
  if ! echo "$REGIONZONE_JSON" | jq -e '.regionzone and (.regionzone | type == "array")' > /dev/null; then
    echo "[ERROR] Failed to parse valid regionzone list."
    echo "$REGIONZONE_JSON" | jq .
  else
    echo "$REGIONZONE_JSON" | jq -r '.regionzone[] | select(.ZoneList != null) | .Name as $region | .ZoneList[] | "'"$PREFIX"'.\($region).\(.Name)"' | while read -r REGION_ID; do
      echo "Deleting connection config: $REGION_ID"
      curl -s -X DELETE "http://$RESTSERVER:1024/spider/connectionconfig/$REGION_ID" \
        -H 'Content-Type: application/json' > /dev/null

      echo "Deleting region: $REGION_ID"
      curl -s -X DELETE "http://$RESTSERVER:1024/spider/region/$REGION_ID" \
        -H 'Content-Type: application/json' > /dev/null
    done
  fi
}

# Function to delete credential
delete_credential() {
  local CREDENTIAL_NAME=$1

  echo "####################################################################"
  echo "## Cloud Credential Deletion"
  echo "####################################################################"

  curl -s -X DELETE "http://$RESTSERVER:1024/spider/credential/$CREDENTIAL_NAME" \
    -H 'Content-Type: application/json' > /dev/null
  echo "Deleted credential: $CREDENTIAL_NAME"
}

# Function to delete driver
delete_driver() {
  local DRIVER_NAME=$1

  echo "####################################################################"
  echo "## Cloud Driver Deletion"
  echo "####################################################################"

  curl -s -X DELETE "http://$RESTSERVER:1024/spider/driver/$DRIVER_NAME" \
    -H 'Content-Type: application/json' > /dev/null
  echo "Deleted driver: $DRIVER_NAME"
}

# Function to display completion message
show_completion() {
  echo "####################################################################"
  echo "## All Deletion Tasks Completed"
  echo "####################################################################"
  echo -e "\n"
}
