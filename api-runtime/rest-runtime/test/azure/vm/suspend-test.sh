RESTSERVER=node12

curl -X GET http://$RESTSERVER:1024/controlvm/vm01?connection_name=azure-config01&action=suspend
