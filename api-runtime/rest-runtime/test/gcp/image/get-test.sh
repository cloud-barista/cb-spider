RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/spider/vmimage/windows-server-2019-dc-v20191112?connection_name=gcp-config01 |json_pp
