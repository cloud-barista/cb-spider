RESTSERVER=localhost

IMAGE_ID=ubuntu16.04
curl -X GET http://$RESTSERVER:1024/vmimage/$IMAGE_ID?connection_name=openstack-config01 |json_pp
