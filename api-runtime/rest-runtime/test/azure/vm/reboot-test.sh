RESTSERVER=localhost

VM_NAME=CBVm
curl -X GET "http://$RESTSERVER:1024/controlvm/$VM_NAME?connection_name=azure-config01&action=reboot"
