source ../setup.env

curl -X GET "http://$RESTSERVER:1024/controlvm/CBVm?connection_name=azure-config01&action=reboot"
