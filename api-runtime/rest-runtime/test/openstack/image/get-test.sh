RESTSERVER=node12

imageId = b3411d30-b054-4c02-8bd1-67cd78aecd63

curl -X GET http://$RESTSERVER:1024/vmimage/$imageId/connection_name=openstack-config01 |json_pp
