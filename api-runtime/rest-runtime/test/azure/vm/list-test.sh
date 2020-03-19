RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/spider/vm?connection_name=azure-config01 |json_pp
