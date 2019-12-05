RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/vmimage?connection_name=gcp-config01 |json_pp