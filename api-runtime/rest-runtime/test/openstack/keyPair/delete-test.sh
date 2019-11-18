RESTSERVER=localhost

KEYPAIR_ID=CB-Keypair
curl -X DELETE http://$RESTSERVER:1024/keypair/$KEYPAIR_ID?connection_name=openstack-config01
