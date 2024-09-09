#!/bin/bash

RESTSERVER=localhost

echo "####################################################################"
echo "## Cloud Driver Info"
echo "####################################################################"

# Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver \
    -H 'Content-Type: application/json' \
    -d '{
        "DriverName": "aws-driver01",
        "ProviderName": "AWS",
        "DriverLibFileName": "aws-driver-v1.0.so"
    }'

echo "####################################################################"
echo "## Cloud Credential Info"
echo "####################################################################"

# Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential \
    -H 'Content-Type: application/json' \
    -d '{
        "CredentialName": "aws-credential01",
        "ProviderName": "AWS",
        "KeyValueInfoList": [
            {"Key": "aws_access_key_id", "Value": "XXXXXXXXXXXXXXXXXXXXXXX"},
            {"Key": "aws_secret_access_key", "Value": "XXXXXXXXXXXXXXXXXXXXXXX"}
        ]
    }'

echo "####################################################################"
echo "## Cloud Region Info"
echo "####################################################################"

# Cloud Region Info
regions=("aws-ohio:us-east-2:us-east-2a"
         "aws-oregon:us-west-2:us-west-2a"
         "aws-singapore:ap-southeast-1:ap-southeast-1a"
         "aws-paris:eu-west-3:eu-west-3a"
         "aws-saopaulo:sa-east-1:sa-east-1a"
         "aws-tokyo:ap-northeast-1:ap-northeast-1a")

for region in "${regions[@]}"; do
    IFS=":" read -r RegionName Region Zone <<< "$region"
    curl -X POST http://$RESTSERVER:1024/spider/region \
        -H 'Content-Type: application/json' \
        -d '{
            "RegionName": "'$RegionName'",
            "ProviderName": "AWS",
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
configs=("aws-ohio-config:aws-ohio"
         "aws-oregon-config:aws-oregon"
         "aws-singapore-config:aws-singapore"
         "aws-paris-config:aws-paris"
         "aws-saopaulo-config:aws-saopaulo"
         "aws-tokyo-config:aws-tokyo")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -X POST http://$RESTSERVER:1024/spider/connectionconfig \
        -H 'Content-Type: application/json' \
        -d '{
            "ConfigName": "'$ConfigName'",
            "ProviderName": "AWS",
            "DriverName": "aws-driver01",
            "CredentialName": "aws-credential01",
            "RegionName": "'$RegionName'"
        }'
done

