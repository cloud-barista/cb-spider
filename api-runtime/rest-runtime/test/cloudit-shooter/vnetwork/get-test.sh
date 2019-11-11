source ../setup.env

VNETWORK_ID=10.0.8.0
curl -X GET http://$RESTSERVER:1024/vnetwork/$VNETWORK_ID?connection_name=cloudit-config01 | json_pp

