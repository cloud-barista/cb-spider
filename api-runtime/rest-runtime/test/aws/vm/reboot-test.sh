RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/controlvm/vm01?connection_name=aws-config01&action=reboot |json_pp
