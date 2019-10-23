RESTSERVER=localhost

IMAGE_ID=fef5366b-688d-4566-a687-736e8fd15032
curl -X GET http://$RESTSERVER:1024/vmimage/$IMAGE_ID?connection_name=openstack-config01 |json_pp
