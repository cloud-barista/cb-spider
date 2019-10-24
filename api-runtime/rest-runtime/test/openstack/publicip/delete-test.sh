RESTSERVER=localhost

PUBLICIP_ID=381a10f8-5831-4822-8388-922673addde4
curl -X DELETE http://$RESTSERVER:1024/publicip/$PUBLICIP_ID?connection_name=openstack-config01
