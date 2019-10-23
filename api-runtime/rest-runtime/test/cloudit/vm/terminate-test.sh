RESTSERVER=192.168.130.8

#vmId => ServerID
curl -X DELETE http://$RESTSERVER:1024/vm/mcb-vm?connection_name=cloudit-config01
