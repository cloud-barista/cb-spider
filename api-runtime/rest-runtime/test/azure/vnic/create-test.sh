RESTSERVER=node12

SECGROUP_ID_1=/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/CB-GROUP/providers/Microsoft.Network/networkSecurityGroups/CB-SecGroup
PUBLICIP_ID=/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/CB-GROUP/providers/Microsoft.Network/publicIPAddresses/CB-PublicIP

curl -X POST http://192.168.130.8:1024/vnic?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{ "Name": "CB-VNic", "VNetName": "CB-Subnet", "SecurityGroupIds": ["/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/CB-GROUP/providers/Microsoft.Network/networkSecurityGroups/CB-SecGroup"], "PublicIPid": "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/CB-GROUP/providers/Microsoft.Network/publicIPAddresses/CB-PublicIP" }'
