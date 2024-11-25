#!/bin/bash

echo "####################################################################"
echo "## Cloud Driver Info"
echo "####################################################################"

# Load sensitive information from .gcp-credential
GCP_CREDENTIAL_FILE="./.gcp-credential"
if [[ -f $GCP_CREDENTIAL_FILE ]]; then
    source $GCP_CREDENTIAL_FILE
else
    echo "Error: Credential file not found at $GCP_CREDENTIAL_FILE"
    exit 1
fi

if [[ -z "$gcp_private_key" || -z "$gcp_project_id" || -z "$gcp_client_email" ]]; then
    echo "Error: Missing one or more required GCP credential variables in credential file."
    exit 1
fi

# Cloud Driver Info
curl -X POST "http://$RESTSERVER:1024/spider/driver" \
    -H 'Content-Type: application/json' \
    -d '{
        "DriverName":"gcp-driver01",
        "ProviderName":"GCP", 
        "DriverLibFileName":"gcp-driver-v1.0.so"
    }'

echo "####################################################################"
echo "## Cloud Credential Info"
echo "####################################################################"

# Cloud Credential Info
curl -X POST "http://$RESTSERVER:1024/spider/credential" \
    -H 'Content-Type: application/json' \
    -d '{
        "CredentialName":"gcp-credential01",
        "ProviderName":"GCP", 
        "KeyValueInfoList": [
            {"Key":"PrivateKey", "Value":"'"$gcp_private_key"'"},
            {"Key":"ProjectID", "Value":"'"$gcp_project_id"'"},
            {"Key":"ClientEmail", "Value":"'"$gcp_client_email"'"}
        ]
    }'

echo "####################################################################"
echo "## Cloud Region Info"
echo "####################################################################"

# Cloud Region Info
regions=("gcp-iowa:us-central1:us-central1-a"
         "gcp-oregon:us-west1:us-west1-a"
         "gcp-london:europe-west2:europe-west2-a"
         "gcp-tokyo:asia-northeast1:asia-northeast1-a"
         "gcp-seoul:asia-northeast3:asia-northeast3-a")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -X POST "http://$RESTSERVER:1024/spider/region" \
        -H 'Content-Type: application/json' \
        -d '{
            "RegionName": "'$RegionName'",
            "ProviderName": "GCP",
            "KeyValueInfoList": [
                {"Key": "Region", "Value": "'$Region'"},
                {"Key": "Zone", "Value": "'$Zone'"}
            ]
        }'
done

echo "####################################################################"
echo "## Cloud Connection Config Info"
echo "####################################################################"

# Cloud Connection Config Info
configs=("gcp-iowa-config:gcp-iowa")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -X POST "http://$RESTSERVER:1024/spider/connectionconfig" \
        -H 'Content-Type: application/json' \
        -d '{
            "ConfigName": "'$ConfigName'",
            "ProviderName": "GCP",
            "DriverName": "gcp-driver01",
            "CredentialName": "gcp-credential01",
            "RegionName": "'$RegionName'"
        }'
done
