source ../setup.env

curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=aws-ohio-config -H 'Content-Type: application/json' -d '{"Name":"CB-VNet-powerkim"}' |json_pp
curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=aws-oregon-config -H 'Content-Type: application/json' -d '{"Name":"CB-VNet-powerkim"}' |json_pp
curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=aws-singapore-config -H 'Content-Type: application/json' -d '{"Name":"CB-VNet-powerkim"}' |json_pp
curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=aws-paris-config -H 'Content-Type: application/json' -d '{"Name":"CB-VNet-powerkim"}' |json_pp
curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=aws-saopaulo-config -H 'Content-Type: application/json' -d '{"Name":"CB-VNet-powerkim"}' |json_pp


curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=aws-tokyo-config -H 'Content-Type: application/json' -d '{"Name":"CB-VNet-powerkim"}' |json_pp
