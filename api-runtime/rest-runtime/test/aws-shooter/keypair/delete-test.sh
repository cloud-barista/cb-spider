RESTSERVER=localhost

curl -X DELETE http://$RESTSERVER:1024/keypair/mcb-keypair-powerkim?connection_name=aws-ohio-config |json_pp
curl -X DELETE http://$RESTSERVER:1024/keypair/mcb-keypair-powerkim?connection_name=aws-oregon-config |json_pp
curl -X DELETE http://$RESTSERVER:1024/keypair/mcb-keypair-powerkim?connection_name=aws-singapore-config |json_pp
curl -X DELETE http://$RESTSERVER:1024/keypair/mcb-keypair-powerkim?connection_name=aws-paris-config |json_pp
curl -X DELETE http://$RESTSERVER:1024/keypair/mcb-keypair-powerkim?connection_name=aws-saopaulo-config |json_pp

curl -X DELETE http://$RESTSERVER:1024/keypair/mcb-keypair-powerkim?connection_name=aws-tokyo-config |json_pp
