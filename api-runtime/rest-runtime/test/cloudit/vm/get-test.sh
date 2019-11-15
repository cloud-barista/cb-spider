RESTSERVER=localhost

VM_NAME=CBVm
#curl -X GET http://$RESTSERVER:1024/vm/$VM_ID?connection_name=cloudit-config01 | json_pp
curl -X GET http://$RESTSERVER:1024/vm/$VM_NAME?connection_name=cloudit-config01 | json_pp
