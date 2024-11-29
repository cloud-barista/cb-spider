#!/bin/bash

echo "####################################################################"
echo "## Cloud Driver Info"
echo "####################################################################"

# Load sensitive information from .alibaba-credential
ALIBABA_CREDENTIAL_FILE="./.alibaba-credential"
if [[ -f $ALIBABA_CREDENTIAL_FILE ]]; then
    source $ALIBABA_CREDENTIAL_FILE
else
    echo "Error: Credential file not found at $ALIBABA_CREDENTIAL_FILE"
    exit 1
fi

if [[ -z "$alibaba_client_id" || -z "$alibaba_client_secret" ]]; then
    echo "Error: Missing one or more required Alibaba credential variables in credential file."
    exit 1
fi

# Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver \
    -H 'Content-Type: application/json' \
    -d '{
        "DriverName":"alibaba-driver01",
        "ProviderName":"ALIBABA", 
        "DriverLibFileName":"alibaba-driver-v1.0.so"
    }'

echo "####################################################################"
echo "## Cloud Credential Info"
echo "####################################################################"

# Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential \
    -H 'Content-Type: application/json' \
    -d '{
        "CredentialName":"alibaba-credential01",
        "ProviderName":"ALIBABA", 
        "KeyValueInfoList": [
            {"Key":"ClientId", "Value":"'"$alibaba_client_id"'"},
            {"Key":"ClientSecret", "Value":"'"$alibaba_client_secret"'"}
        ]
    }'

echo "####################################################################"
echo "## Cloud Region Info"
echo "####################################################################"

# Cloud Region Info
regions=("alibaba-beijing:cn-beijing:cn-beijing-f"
         "alibaba-ulanqab:cn-wulanchabu:cn-wulanchabu-1a"
         "alibaba-london:eu-west-1:eu-west-1a"
         "alibaba-tokyo:ap-northeast-1:ap-northeast-1a"
         "alibaba-singapore:ap-southeast-1:ap-southeast-1c"
         "alibaba-hongkong:cn-hongkong:cn-hongkong-c")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -X POST http://$RESTSERVER:1024/spider/region \
        -H 'Content-Type: application/json' \
        -d '{
            "RegionName": "'$RegionName'",
            "ProviderName": "ALIBABA",
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
configs=("alibaba-beijing-config:alibaba-beijing")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -X POST http://$RESTSERVER:1024/spider/connectionconfig \
        -H 'Content-Type: application/json' \
        -d '{
            "ConfigName": "'$ConfigName'",
            "ProviderName": "ALIBABA",
            "DriverName": "alibaba-driver01",
            "CredentialName": "alibaba-credential01",
            "RegionName": "'$RegionName'"
        }'
done
