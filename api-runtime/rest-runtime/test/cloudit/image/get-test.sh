RESTSERVER=localhost

IMAGE_ID=CentOS-7
curl -X GET http://$RESTSERVER:1024/vmimage/$IMAGE_ID?connection_name=cloudit-config01 | json_pp
