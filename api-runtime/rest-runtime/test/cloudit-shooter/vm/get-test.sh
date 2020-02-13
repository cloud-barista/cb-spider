source ../setup.env

curl -sX GET http://$RESTSERVER:1024/vm/vm-powerkim01?connection_name=cloudit-config01 |json_pp
