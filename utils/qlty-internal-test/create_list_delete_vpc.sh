SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


echo "####################################################################"
echo "## VPC: Create"
echo "####################################################################"
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX POST http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "vpc-01", "IPv4_CIDR": "192.168.0.0/16", "SubnetInfoList": [ { "Name": "subnet-01", "IPv4_CIDR": "192.168.1.0/24"} ] } }'
echo "#-----------------------------"


echo "####################################################################"
echo "## VPC: Get"
echo "####################################################################"
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX GET http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp


echo "####################################################################"
echo "## VPC: Delete"
echo "####################################################################"
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX DELETE http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
