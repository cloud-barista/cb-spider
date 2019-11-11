source ../setup.env

curl -X DELETE "http://$RESTSERVER:1024/vm/CBVm?connection_name=azure-config01"
