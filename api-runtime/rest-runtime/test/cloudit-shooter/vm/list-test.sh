source ../setup.env

curl -sX GET http://$RESTSERVER:1024/vm?connection_name=cloudit-config01 | json_pp
