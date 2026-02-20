#!/bin/bash
API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD


echo "####################################################################"
echo "## Cloud Connection Config Info Deletion"
echo "####################################################################"

# Cloud Connection Config Info Deletion
configs=("tencent-beijing3-config:tencent-beijing-3")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -u $API_USERNAME:$API_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/connectionconfig/"$ConfigName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Region Info Deletion"
echo "####################################################################"

# Cloud Region Info Deletion
regions=("tencent-guangzhou-3:ap-guangzhou:ap-guangzhou-6"
         "tencent-beijing-3:ap-beijing:ap-beijing-3"
         "tencent-seoul-1:ap-seoul:ap-seoul-1"
         "tencent-tokyo-1:ap-tokyo:ap-tokyo-1"
         "tencent-frankfurt-1:eu-frankfurt:eu-frankfurt-1")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -u $API_USERNAME:$API_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/region/"$RegionName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Credential Info Deletion"
echo "####################################################################"

# Cloud Credential Info Deletion
curl -u $API_USERNAME:$API_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/credential/tencent-credential01 \
    -H 'Content-Type: application/json'

echo "####################################################################"
echo "## Cloud Driver Info Deletion"
echo "####################################################################"

# Cloud Driver Info Deletion
curl -u $API_USERNAME:$API_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/driver/tencent-driver01 \
    -H 'Content-Type: application/json'

