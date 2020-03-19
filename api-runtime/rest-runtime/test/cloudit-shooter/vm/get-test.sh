source ../setup.env

curl -sX GET http://$RESTSERVER:1024/spider/vm/vm-powerkim01?connection_name=cloudit-config01 |json_pp
