RESTSERVER=localhost

SECURITYGROUP_ID=CB-SecGroup
curl -X GET http://$RESTSERVER:1024/securitygroup/$SECURITYGROUP_ID?connection_name=openstack-config01 |json_pp
