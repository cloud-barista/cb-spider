RESTSERVER=localhost

curl -X GET "http://$RESTSERVER:1024/spider/controlvm/vm01?connection_name=gcp-config01&action=suspend"