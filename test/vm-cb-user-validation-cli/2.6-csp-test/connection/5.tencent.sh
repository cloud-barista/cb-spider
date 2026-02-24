#!/bin/bash
SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


echo "####################################################################"
echo "## Cloud Driver Info"
echo "####################################################################"

# Load sensitive information from .tencent-credential
TENCENT_CREDENTIAL_FILE="./.tencent-credential"
if [[ -f $TENCENT_CREDENTIAL_FILE ]]; then
    source $TENCENT_CREDENTIAL_FILE
else
    echo "Error: Credential file not found at $TENCENT_CREDENTIAL_FILE"
    exit 1
fi

if [[ -z "$tencent_secret_id" || -z "$tencent_secret_key" ]]; then
    echo "Error: Missing one or more required Tencent credential variables in credential file."
    exit 1
fi

# Cloud Driver Info
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X POST http://$RESTSERVER:1024/spider/driver \
    -H 'Content-Type: application/json' \
    -d '{
        "DriverName":"tencent-driver01",
        "ProviderName":"TENCENT", 
        "DriverLibFileName":"tencent-driver-v1.0.so"
    }'

echo "####################################################################"
echo "## Cloud Credential Info"
echo "####################################################################"

# Cloud Credential Info
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X POST http://$RESTSERVER:1024/spider/credential \
    -H 'Content-Type: application/json' \
    -d '{
        "CredentialName":"tencent-credential01",
        "ProviderName":"TENCENT", 
        "KeyValueInfoList": [
            {"Key":"SecretId", "Value":"'"$tencent_secret_id"'"},
            {"Key":"SecretKey", "Value":"'"$tencent_secret_key"'"}
        ]
    }'

echo "####################################################################"
echo "## Cloud Region Info"
echo "####################################################################"

# Cloud Region Info
regions=("tencent-guangzhou-3:ap-guangzhou:ap-guangzhou-6"
         "tencent-beijing-3:ap-beijing:ap-beijing-3"
         "tencent-seoul-1:ap-seoul:ap-seoul-1"
         "tencent-tokyo-1:ap-tokyo:ap-tokyo-1"
         "tencent-frankfurt-1:eu-frankfurt:eu-frankfurt-1")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X POST http://$RESTSERVER:1024/spider/region \
        -H 'Content-Type: application/json' \
        -d '{
            "RegionName": "'$RegionName'",
            "ProviderName": "TENCENT",
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
configs=("tencent-beijing3-config:tencent-beijing-3")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X POST http://$RESTSERVER:1024/spider/connectionconfig \
        -H 'Content-Type: application/json' \
        -d '{
            "ConfigName": "'$ConfigName'",
            "ProviderName": "TENCENT",
            "DriverName": "tencent-driver01",
            "CredentialName": "tencent-credential01",
            "RegionName": "'$RegionName'"
        }'
done
