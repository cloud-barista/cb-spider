#!/bin/bash
SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


echo "####################################################################"
echo "## Cloud Connection Config Info Deletion"
echo "####################################################################"

# Cloud Connection Config Info Deletion
configs=("gcp-iowa-config:gcp-iowa")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE "http://$RESTSERVER:1024/spider/connectionconfig/$ConfigName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Region Info Deletion"
echo "####################################################################"

# Cloud Region Info Deletion
regions=("gcp-iowa:us-central1:us-central1-a"
         "gcp-oregon:us-west1:us-west1-a"
         "gcp-london:europe-west2:europe-west2-a"
         "gcp-tokyo:asia-northeast1:asia-northeast1-a"
         "gcp-seoul:asia-northeast3:asia-northeast3-a")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE "http://$RESTSERVER:1024/spider/region/$RegionName" \
        -H 'Content-Type: application/json'
done

echo "####################################################################"
echo "## Cloud Credential Info Deletion"
echo "####################################################################"

# Cloud Credential Info Deletion
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE "http://$RESTSERVER:1024/spider/credential/gcp-credential01" \
    -H 'Content-Type: application/json'

echo "####################################################################"
echo "## Cloud Driver Info Deletion"
echo "####################################################################"

# Cloud Driver Info Deletion
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -X DELETE "http://$RESTSERVER:1024/spider/driver/gcp-driver01" \
    -H 'Content-Type: application/json'

