RESTSERVER=localhost

curl -X DELETE http://$RESTSERVER:1024/vm/vm01?connection_name=aws-config01 |json_pp
