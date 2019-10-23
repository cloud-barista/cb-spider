RESTSERVER=localhost

SECURITYGROUP_ID=8d9fd96f-61da-4e4f-9370-4f363bf838b8
curl -X DELETE http://$RESTSERVER:1024/securitygroup/$SECURITYGROUP_ID?connection_name=openstack-config01
