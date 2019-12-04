RESTSERVER=localhost

VM_NAME=CBVm
curl -X DELETE "http://$RESTSERVER:1024/vm/$VM_NAME?connection_name=azure-config01"
