source ../setup.env

PUBLICIP_ADDR=182.252.135.44
curl -X DELETE http://$RESTSERVER:1024/publicip/$PUBLICIP_ADDR?connection_name=cloudit-config01

