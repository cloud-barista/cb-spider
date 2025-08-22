#!/bin/bash
source ./1.export.env

echo "curl -v -s -X POST $OS_NETWORK_API/Network"
curl -v -s -X POST $OS_NETWORK_API/Network --header "X-Auth-Token: $OS_TOKEN" 'Content-Type: application/json' \
--data-raw '
{ 
	"name": "test-tier-1",
	"zone": "DX-M1",
	"type": "tier",
	"usercustom": "y",
	"detail": {
		"cidr": "172.25.7.0/24",
		"startip": "172.25.7.11",
		"endip": "172.25.7.120",
		"lbstartip": "172.25.7.140",
		"lbendip": "172.25.7.190",
		"bmstartip": "172.25.7.201",
		"bmendip": "172.25.7.250",
		"gateway": "172.25.7.1"
	}
}' ; echo 
echo -e "\n"

# String 입력시 양쪽에 [] 기호는 없이 입력해야함.
