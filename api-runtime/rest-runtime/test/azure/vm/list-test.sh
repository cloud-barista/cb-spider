RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/vm?connection_name=azure-config01 |json_pp
