source ../setup.env

PUBLICIP_ADDR=182.252.135.44
curl -X GET http://$RESTSERVER:1024/publicip/$PUBLICIP_ADDR?connection_name=cloudit-config01 | json_pp

