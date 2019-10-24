RESTSERVER=localhost

IMAGE_ID=a846af3b-5d80-4182-b38e-5501ad9f78f4
curl -X GET http://$RESTSERVER:1024/vmimage/$IMAGE_ID?connection_name=cloudit-config01 | json_pp
