API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD

echo "####################################################################"
echo "## Cloud Driver Info"
echo "####################################################################"

# Load sensitive information from .aws-credential
AWS_CREDENTIAL_FILE="./.aws-credential"
if [[ -f $AWS_CREDENTIAL_FILE ]]; then
    source $AWS_CREDENTIAL_FILE
else
    echo "Error: Credential file not found at $AWS_CREDENTIAL_FILE"
    exit 1
fi

if [[ -z "$aws_access_key_id" || -z "$aws_secret_access_key" ]]; then
    echo "Error: Missing aws_access_key_id or aws_secret_access_key in credential file."
    exit 1
fi

# Cloud Driver Info
curl -u $API_USERNAME:$API_PASSWORD -X POST http://$RESTSERVER:1024/spider/driver \
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
curl -u $API_USERNAME:$API_PASSWORD -X POST http://$RESTSERVER:1024/spider/credential \
    -H 'Content-Type: application/json' \
    -d '{
        "CredentialName": "aws-credential01",
        "ProviderName": "AWS",
        "KeyValueInfoList": [
            {"Key": "aws_access_key_id", "Value": "'"$aws_access_key_id"'"},
            {"Key": "aws_secret_access_key", "Value": "'"$aws_secret_access_key"'"}
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
    curl -u $API_USERNAME:$API_PASSWORD -X POST http://$RESTSERVER:1024/spider/region \
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
configs=("aws-config01:aws-ohio")

for config in "${configs[@]}"; do
    IFS=":" read -r ConfigName RegionName <<< "$config"
    curl -u $API_USERNAME:$API_PASSWORD -X POST http://$RESTSERVER:1024/spider/connectionconfig \
        -H 'Content-Type: application/json' \
        -d '{
            "ConfigName": "'$ConfigName'",
            "ProviderName": "AWS",
            "DriverName": "aws-driver01",
            "CredentialName": "aws-credential01",
            "RegionName": "'$RegionName'"
        }'
done