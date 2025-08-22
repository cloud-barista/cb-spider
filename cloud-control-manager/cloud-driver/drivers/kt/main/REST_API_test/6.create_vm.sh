#!/bin/bash
source ./1.export.env

echo "curl -v -s -X POST $OS_COMPUTE_API/servers"
curl -v -s -X POST $OS_COMPUTE_API/servers --header "X-Auth-Token: $OS_TOKEN" 'Content-Type: application/json' \
--data-raw '
{
	"server": {
		"name": "kt-vm-1",
		"key_name": "KeyPair-For-Test",
		"flavorRef": "f4556759-9d9e-4e43-8448-b2fe104e5a11",
		"availability_zone": "DX-M1",
		"networks": [
			{
				"uuid": "3eff0710-269f-45f5-8ca9-6c0fb90c5b2b"
			}
		],
		"block_device_mapping_v2": [
			{
				"destination_type": "volume",
				"boot_index": "0",
				"source_type": "image",
				"volume_size": "50",
				"uuid": "63ce2b32-1364-4ba5-ad02-78e7251e6e47"
			}
		]
	}
}' ; echo 
echo -e "\n"

# String 입력시 양쪽에 [] 기호는 없이 입력해야함.
