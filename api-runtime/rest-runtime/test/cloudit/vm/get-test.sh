RESTSERVER=localhost

VM_NAME=CBVm
curl -X GET http://$RESTSERVER:1024/spider/vm/$VM_NAME?connection_name=cloudit-config01 | json_pp
