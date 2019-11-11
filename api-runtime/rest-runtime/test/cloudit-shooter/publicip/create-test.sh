source ../setup.env

curl -X POST http://$RESTSERVER:1024/publicip?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{ "Name": "publicipt01-powerkim", "KeyValueList": [{"Key":"PrivateIP", "Value":"10.0.0.2"}]}' |json_pp

