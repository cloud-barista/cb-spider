RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/vm?connection_name=cloudit-config01 | json_pp
