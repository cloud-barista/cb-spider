RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/vmstatus?connection_name=aws-config01 |json_pp
