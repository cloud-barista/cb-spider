source ../setup.env

curl -X GET http://$RESTSERVER:1024/vmstatus?connection_name=azure-config01
