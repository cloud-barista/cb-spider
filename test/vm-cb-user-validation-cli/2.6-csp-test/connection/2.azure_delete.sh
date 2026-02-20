#!/bin/bash
API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD


echo "####################################################################"
echo "## Cloud Connection Config Info Deletion"
echo "####################################################################"

# Cloud Connection Config Info Deletion
configs=("azure-northeu-config:azure-northeu")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -u $API_USERNAME:$API_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/connectionconfig/"$ConfigName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Region Info Deletion"
echo "####################################################################"

# Cloud Region Info Deletion
regions=("azure-northeu:northeurope:1"
         "azure-eastus:eastus:1"
         "azure-westus:westus:1"
         "azure-japanwest:japanwest:1"
         "azure-koreacentral:koreacentral:1")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -u $API_USERNAME:$API_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/region/"$RegionName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Credential Info Deletion"
echo "####################################################################"

# Cloud Credential Info Deletion
curl -u $API_USERNAME:$API_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/credential/azure-credential01 \
    -H 'Content-Type: application/json'

echo "####################################################################"
echo "## Cloud Driver Info Deletion"
echo "####################################################################"

# Cloud Driver Info Deletion
curl -u $API_USERNAME:$API_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/driver/azure-driver01 \
    -H 'Content-Type: application/json'

