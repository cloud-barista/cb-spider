#!/bin/bash
SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


echo "####################################################################"
echo "## Cloud Connection Config Info Deletion"
echo "####################################################################"

# Cloud Connection Config Info Deletion
configs=("alibaba-beijing-config:alibaba-beijing")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/connectionconfig/"$ConfigName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Region Info Deletion"
echo "####################################################################"

# Cloud Region Info Deletion
regions=("alibaba-beijing:cn-beijing:cn-beijing-f"
         "alibaba-ulanqab:cn-wulanchabu:cn-wulanchabu-1a"
         "alibaba-london:eu-west-1:eu-west-1a"
         "alibaba-tokyo:ap-northeast-1:ap-northeast-1a"
         "alibaba-singapore:ap-southeast-1:ap-southeast-1c"
         "alibaba-hongkong:cn-hongkong:cn-hongkong-c")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/region/"$RegionName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Credential Info Deletion"
echo "####################################################################"

# Cloud Credential Info Deletion
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/credential/alibaba-credential01 \
    -H 'Content-Type: application/json'

echo "####################################################################"
echo "## Cloud Driver Info Deletion"
echo "####################################################################"

# Cloud Driver Info Deletion
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/driver/alibaba-driver01 \
    -H 'Content-Type: application/json'

