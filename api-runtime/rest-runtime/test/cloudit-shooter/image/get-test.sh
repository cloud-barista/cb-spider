source ../setup.env

curl -X GET http://$RESTSERVER:1024/vmimage/$CLOUDIT_IMG_ID1?connection_name=cloudit-config01 | json_pp
curl -X GET http://$RESTSERVER:1024/vmimage/$CLOUDIT_IMG_ID2?connection_name=cloudit-config01 | json_pp
curl -X GET http://$RESTSERVER:1024/vmimage/$CLOUDIT_IMG_ID3?connection_name=cloudit-config01 | json_pp
