RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/spider/vnetwork?connection_name=openstack-config01 |json_pp
