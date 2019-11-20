source ../setup.env

curl -sX DELETE "http://$RESTSERVER:1024/vm/vm-powerkim01?connection_name=cloudit-config01"
