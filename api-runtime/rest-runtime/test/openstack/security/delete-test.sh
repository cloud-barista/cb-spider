RESTSERVER=localhost

SECURITYGROUP_ID=CB-SecGroup
curl -X DELETE http://$RESTSERVER:1024/securitygroup/$SECURITYGROUP_ID?connection_name=openstack-config01
