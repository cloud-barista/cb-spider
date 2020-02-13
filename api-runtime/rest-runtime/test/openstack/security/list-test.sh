RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=openstack-config01 |json_pp
