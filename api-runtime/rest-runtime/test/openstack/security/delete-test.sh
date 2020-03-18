RESTSERVER=localhost

SECURITY_NAME=CB-SecGroup
curl -X DELETE http://$RESTSERVER:1024/spider/securitygroup/$SECURITY_NAME?connection_name=openstack-config01
