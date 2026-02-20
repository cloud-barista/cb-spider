#!/bin/bash
API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD


echo "####################################################################"
echo "## Cloud Driver Info"
echo "####################################################################"

# Load sensitive information from .azure-credential
AZURE_CREDENTIAL_FILE="./.azure-credential"
if [[ -f $AZURE_CREDENTIAL_FILE ]]; then
    source $AZURE_CREDENTIAL_FILE
else
    echo "Error: Credential file not found at $AZURE_CREDENTIAL_FILE"
    exit 1
fi

if [[ -z "$azure_client_id" || -z "$azure_client_secret" || -z "$azure_tenant_id" || -z "$azure_subscription_id" ]]; then
    echo "Error: Missing one or more required Azure credential variables in credential file."
    exit 1
fi

# Cloud Driver Info
curl -u $API_USERNAME:$API_PASSWORD -X POST http://$RESTSERVER:1024/spider/driver \
    -H 'Content-Type: application/json' \
    -d '{
        "DriverName":"azure-driver01",
        "ProviderName":"AZURE", 
        "DriverLibFileName":"azure-driver-v1.0.so"
    }'

echo "####################################################################"
echo "## Cloud Credential Info"
echo "####################################################################"

# Cloud Credential Info
curl -u $API_USERNAME:$API_PASSWORD -X POST http://$RESTSERVER:1024/spider/credential \
    -H 'Content-Type: application/json' \
    -d '{
        "CredentialName":"azure-credential01",
        "ProviderName":"AZURE", 
        "KeyValueInfoList": [
            {"Key":"ClientId", "Value":"'"$azure_client_id"'"},
            {"Key":"ClientSecret", "Value":"'"$azure_client_secret"'"},
            {"Key":"TenantId", "Value":"'"$azure_tenant_id"'"},
            {"Key":"SubscriptionId", "Value":"'"$azure_subscription_id"'"}
        ]
    }'

echo "####################################################################"
echo "## Cloud Region Info"
echo "####################################################################"

# Cloud Region Info
regions=("azure-northeu:northeurope:1"
         "azure-eastus:eastus:1"
         "azure-westus:westus:1"
         "azure-japanwest:japanwest:1"
         "azure-koreacentral:koreacentral:1")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -u $API_USERNAME:$API_PASSWORD -X POST http://$RESTSERVER:1024/spider/region \
        -H 'Content-Type: application/json' \
        -d '{
            "RegionName": "'$RegionName'",
            "ProviderName": "AZURE",
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
configs=("azure-northeu-config:azure-northeu")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -u $API_USERNAME:$API_PASSWORD -X POST http://$RESTSERVER:1024/spider/connectionconfig \
        -H 'Content-Type: application/json' \
        -d '{
            "ConfigName": "'$ConfigName'",
            "ProviderName": "AZURE",
            "DriverName": "azure-driver01",
            "CredentialName": "azure-credential01",
            "RegionName": "'$RegionName'"
        }'
done
