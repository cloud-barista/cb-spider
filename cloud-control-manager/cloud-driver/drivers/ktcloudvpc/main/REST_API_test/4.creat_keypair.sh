#!/bin/bash
source ./1.export.env

echo "curl -v -s -X POST $OS_COMPUTE_API/os-keypairs"
curl -v -s -X POST $OS_COMPUTE_API/os-keypairs --header "X-Auth-Token: $OS_TOKEN" 'Content-Type: application/json' \
--data-raw '
{
	"keypair": {
		"name": "KeyPair-For-Test"
	}
}' ; echo 
echo -e "\n"

# String 입력시 양쪽에 [] 기호는 없이 입력해야함.
