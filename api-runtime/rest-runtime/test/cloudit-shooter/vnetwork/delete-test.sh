source ../setup.env

VNETWORK_ID=CB-VNet-powerkim
curl -sX DELETE http://$RESTSERVER:1024/spider/vnetwork/$VNETWORK_ID?connection_name=cloudit-config01 |json_pp
