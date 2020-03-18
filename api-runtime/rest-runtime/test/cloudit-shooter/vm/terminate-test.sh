source ../setup.env

curl -sX DELETE "http://$RESTSERVER:1024/spider/vm/vm-powerkim01?connection_name=cloudit-config01"
