RESTSERVER=localhost

VNETWORK_ID=CB-Subnet
curl -X GET http://$RESTSERVER:1024/vnetwork/$VNETWORK_ID?connection_name=openstack-config01 |json_pp
