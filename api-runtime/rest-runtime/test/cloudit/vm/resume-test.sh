RESTSERVER=localhost

VM_NAME=CBVm
curl -X GET "http://$RESTSERVER:1024/controlvm/$VM_NAME?connection_name=cloudit-config01&action=resume"
