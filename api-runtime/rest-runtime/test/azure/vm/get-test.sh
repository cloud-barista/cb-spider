RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/vm/CBVm?connection_name=azure-config01 |json_pp
