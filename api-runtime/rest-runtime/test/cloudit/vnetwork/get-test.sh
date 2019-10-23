RESTSERVER=192.168.130.8

# mcb-test-vnet -> vNetworkId 변경 필요
curl -X GET http://$RESTSERVER:1024/vnetwork/mcb-test-vnet?connection_name=cloudit-config01
