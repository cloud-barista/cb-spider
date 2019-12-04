RESTSERVER=localhost

VNETWORK_NAME=CB-Subnet
curl -X GET http://$RESTSERVER:1024/vnetwork/$VNETWORK_NAME?connection_name=azure-config01  |json_pp
