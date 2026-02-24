#!/bin/bash
SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


echo "####################################################################"
echo "## Cloud Connection Config Info Deletion"
echo "####################################################################"

# Cloud Connection Config Info Deletion
configs=("aws-config01:aws-ohio")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/connectionconfig/"$ConfigName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Region Info Deletion"
echo "####################################################################"

# Cloud Region Info Deletion
regions=("aws-ohio:us-east-2:us-east-2a"
         "aws-oregon:us-west-2:us-west-2a"
         "aws-singapore:ap-southeast-1:ap-southeast-1a"
         "aws-paris:eu-west-3:eu-west-3a"
         "aws-saopaulo:sa-east-1:sa-east-1a"
         "aws-tokyo:ap-northeast-1:ap-northeast-1a")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/region/"$RegionName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Credential Info Deletion"
echo "####################################################################"

# Cloud Credential Info Deletion
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/credential/aws-credential01 \
    -H 'Content-Type: application/json'

echo "####################################################################"
echo "## Cloud Driver Info Deletion"
echo "####################################################################"

# Cloud Driver Info Deletion
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE http://$RESTSERVER:1024/spider/driver/aws-driver01 \
    -H 'Content-Type: application/json'

