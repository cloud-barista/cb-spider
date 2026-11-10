#!/bin/bash
#
# Common functions for registering/deleting CB-Spider connection info
# (driver, credential, region, connection config) used by 2.10-csp-test.
#
# - Credentials are loaded from $HOME/.cb-spider/.<csp>-credential
#   (same as utils/default-connection-helper). See ./.<csp>-credential.sample
# - Each <N>.<csp>.sh defines the variables below and calls: run_conn "$@"
#     PROVIDER            ex) AWS
#     DRIVER_NAME         ex) aws-driver01
#     DRIVER_LIB          ex) aws-driver-v1.0.so
#     CREDENTIAL_NAME     ex) aws-credential01
#     CREDENTIAL_FILE     ex) $CRED_DIR/.aws-credential
#     REQUIRED_VARS       array of variable names required in CREDENTIAL_FILE
#     REGIONS             array of "RegionName:Region[:Zone]"
#     CONFIGS             array of "ConfigName:RegionName"
#     credential_key_values()  function that echoes the KeyValueInfoList JSON array
# - <N>.<csp>_delete.sh calls "<N>.<csp>.sh delete".

SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD
RESTSERVER=${RESTSERVER:-localhost}
SPIDER_PORT=${SPIDER_PORT:-1024}
CRED_DIR=${CRED_DIR:-$HOME/.cb-spider}

SPIDER_URL="http://$RESTSERVER:$SPIDER_PORT/spider"

print_title() {
  echo "####################################################################"
  echo "## $1"
  echo "####################################################################"
}

# usage: load_credential
load_credential() {
  if [[ ! -f "$CREDENTIAL_FILE" ]]; then
    echo "Error: Credential file not found at $CREDENTIAL_FILE"
    echo "       (cf. $(pwd)/.$(basename "$CREDENTIAL_FILE" | sed 's/^\.//').sample)"
    exit 1
  fi
  source "$CREDENTIAL_FILE"

  local missing=()
  for var in "${REQUIRED_VARS[@]}"; do
    [[ -z "${!var}" ]] && missing+=("$var")
  done
  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Error: Missing required variable(s) in $CREDENTIAL_FILE: ${missing[*]}"
    exit 1
  fi

  # optional: S3 (Object Storage) keys; S3_VARS is set only for CSPs that need separate S3 keys
  local s3_missing=()
  for var in "${S3_VARS[@]}"; do
    [[ -z "${!var}" ]] && s3_missing+=("$var")
  done
  if [[ ${#s3_missing[@]} -gt 0 ]]; then
    echo "[WARN] S3 key(s) empty in $CREDENTIAL_FILE: ${s3_missing[*]} => Object Storage(S3) test will fail"
  fi
}

# usage: spider_post <resource> <json> <label>
spider_post() {
  local RESOURCE=$1 BODY=$2 LABEL=$3
  local RESP HTTP_CODE RES_BODY

  RESP=$(curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -s -w '\n%{http_code}' -X POST \
    "$SPIDER_URL/$RESOURCE" -H 'Content-Type: application/json' -d "$BODY")
  HTTP_CODE=$(echo "$RESP" | tail -n1)
  RES_BODY=$(echo "$RESP" | sed '$d')

  if [[ "$HTTP_CODE" == 2* ]]; then
    echo "Registered $LABEL"
  else
    echo "[FAIL] $LABEL (HTTP $HTTP_CODE) $(echo "$RES_BODY" | jq -r '.message // empty' 2>/dev/null || echo "$RES_BODY")"
  fi
}

# usage: spider_delete <resource-path> <label>
spider_delete() {
  local RESOURCE_PATH=$1 LABEL=$2
  local RESP HTTP_CODE RES_BODY

  # Spider returns 200 even when deleting a non-existent resource, so check existence first
  HTTP_CODE=$(curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -s -o /dev/null -w '%{http_code}' -X GET \
    "$SPIDER_URL/$RESOURCE_PATH" -H 'accept: application/json')
  if [[ "$HTTP_CODE" != 2* ]]; then
    echo "[SKIP] $LABEL (not found)"
    return 0
  fi

  RESP=$(curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -s -w '\n%{http_code}' -X DELETE \
    "$SPIDER_URL/$RESOURCE_PATH" -H 'Content-Type: application/json')
  HTTP_CODE=$(echo "$RESP" | tail -n1)
  RES_BODY=$(echo "$RESP" | sed '$d')

  if [[ "$HTTP_CODE" == 2* ]]; then
    echo "Deleted $LABEL"
  else
    echo "[FAIL] $LABEL (HTTP $HTTP_CODE) $(echo "$RES_BODY" | jq -r '.message // empty' 2>/dev/null || echo "$RES_BODY")"
  fi
}

register_all() {
  load_credential

  print_title "Cloud Driver Info ($PROVIDER)"
  spider_post "driver" '{
      "DriverName": "'"$DRIVER_NAME"'",
      "ProviderName": "'"$PROVIDER"'",
      "DriverLibFileName": "'"$DRIVER_LIB"'"
    }' "driver: $DRIVER_NAME"

  print_title "Cloud Credential Info ($PROVIDER)"
  spider_post "credential" '{
      "CredentialName": "'"$CREDENTIAL_NAME"'",
      "ProviderName": "'"$PROVIDER"'",
      "KeyValueInfoList": '"$(credential_key_values)"'
    }' "credential: $CREDENTIAL_NAME"

  print_title "Cloud Region Info ($PROVIDER)"
  local RegionName Region Zone KV
  for region in "${REGIONS[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    KV='{"Key": "Region", "Value": "'"$Region"'"}'
    [[ -n "$Zone" ]] && KV+=', {"Key": "Zone", "Value": "'"$Zone"'"}'
    spider_post "region" '{
        "RegionName": "'"$RegionName"'",
        "ProviderName": "'"$PROVIDER"'",
        "KeyValueInfoList": ['"$KV"']
      }' "region: $RegionName"
  done

  print_title "Cloud Connection Config Info ($PROVIDER)"
  local ConfigName
  for config in "${CONFIGS[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    spider_post "connectionconfig" '{
        "ConfigName": "'"$ConfigName"'",
        "ProviderName": "'"$PROVIDER"'",
        "DriverName": "'"$DRIVER_NAME"'",
        "CredentialName": "'"$CREDENTIAL_NAME"'",
        "RegionName": "'"$RegionName"'"
      }' "connection config: $ConfigName"
  done
}

delete_all() {
  print_title "Cloud Connection Config Info Deletion ($PROVIDER)"
  local ConfigName RegionName
  for config in "${CONFIGS[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    spider_delete "connectionconfig/$ConfigName" "connection config: $ConfigName"
  done

  print_title "Cloud Region Info Deletion ($PROVIDER)"
  for region in "${REGIONS[@]}"; do
    IFS=":" read -r RegionName _ <<< "$region"
    spider_delete "region/$RegionName" "region: $RegionName"
  done

  print_title "Cloud Credential Info Deletion ($PROVIDER)"
  spider_delete "credential/$CREDENTIAL_NAME" "credential: $CREDENTIAL_NAME"

  print_title "Cloud Driver Info Deletion ($PROVIDER)"
  spider_delete "driver/$DRIVER_NAME" "driver: $DRIVER_NAME"
}

# usage: run_conn [register|delete]
run_conn() {
  case "${1:-register}" in
    register) register_all ;;
    delete)   delete_all ;;
    *) echo "usage: $0 [register|delete]"; exit 1 ;;
  esac
}
