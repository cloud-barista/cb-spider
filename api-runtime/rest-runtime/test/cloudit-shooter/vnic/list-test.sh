source ../setup.env

curl -X GET http://$RESTSERVER:1024/vnic?connection_name=cloudit-config01 | json_pp
