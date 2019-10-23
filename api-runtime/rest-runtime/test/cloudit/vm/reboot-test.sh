RESTSERVER=192.168.130.8

#vmId => ServerID
curl -X GET http://$RESTSERVER:1024/controlvm/mcb-vm?connection_name=cloudit-config01&action=reboot
