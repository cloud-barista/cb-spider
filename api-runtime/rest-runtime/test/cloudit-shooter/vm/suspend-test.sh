source ../setup.env

curl -sX GET "http://$RESTSERVER:1024/spider/controlvm/vm-powerkim01?connection_name=cloudit-config01&action=suspend"
