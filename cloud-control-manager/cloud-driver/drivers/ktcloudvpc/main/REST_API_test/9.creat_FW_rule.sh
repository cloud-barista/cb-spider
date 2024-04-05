#!/bin/bash
source ./1.export.env

echo "curl -v -s -X POST $OS_NETWORK_API/Firewall"
curl -v -s -X POST $OS_NETWORK_API/Firewall --header "X-Auth-Token: $OS_TOKEN" 'Content-Type: application/json' \
--data-raw '
{ 	
	"action": "allow",
	"srcnetworkid": "34bacd6f-0683-42a2-b576-712c07ed1bcc",
	"srcip": "172.25.6.99/32",
	"protocol": "ICMP",
	"dstnetworkid": "bf9ee5e0-e0b4-4dfc-bc59-ba9498f75efa",
	"dstip": "0.0.0.0/0",
	"startport": "1",
	"endport": "65535",
	"srcnat": "true",
}' ; echo 
echo -e "\n"

# String 입력시 양쪽에 [] 기호는 없이 입력해야함.

# $$$$$ ICMP는 startport, endport parameter 아예 없이



	#"action": "allow",
	#"srcnetworkid": "bf9ee5e0-e0b4-4dfc-bc59-ba9498f75efa",
	#"srcip": "0.0.0.0/0",
	#"protocol": "ICMP",
	#"dstnetworkid": "34bacd6f-0683-42a2-b576-712c07ed1bcc",
	#"dstip": "172.25.6.49/32",
	#"srcnat": "false",


# "srcnat": "true",
# "srcnat": true,	

# "srcnet": "true",	
# "srcnet": true,	

# "srcnetwork": true,
# "srcinterfacename":	"000",

	# "srcinterfacename":	"kt-dx-subnet-1",
	# "dstinterfacename":	"external",


	#	"srcnetwork": {
	#	"srcinterfacename": "kt-dx-subnet-1",
	# }
	
	# "srcip": "172.25.6.1/24",
	# "srcip": "172.25.6.49/32"