source ../setup.env

curl -X GET http://$RESTSERVER:1024/vmstatus/CBVm?connection_name=azure-config01
