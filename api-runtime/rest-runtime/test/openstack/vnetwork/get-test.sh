RESTSERVER=localhost

VNETWORK_ID=93783b70-92ec-47ce-9739-2bdc6df614eb
curl -X GET http://$RESTSERVER:1024/vnetwork/$VNETWORK_ID?connection_name=openstack-config01
