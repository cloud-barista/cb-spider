RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/vmstatus/vm01?connection_name=aws-config01 |json_pp
