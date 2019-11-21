source ../setup.env

curl -sX GET http://$RESTSERVER:1024/vmimage/$CLOUDIT_IMG_ID1?connection_name=cloudit-config01 | json_pp
