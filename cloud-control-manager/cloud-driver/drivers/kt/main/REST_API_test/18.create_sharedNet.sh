#!/bin/bash
source ./1.export.env

### 이 script로 Shared Network 생성 성공했음.
echo "### OS_NAS_API - Creatge Shared Network"
echo "curl -v -s -X POST $OS_NAS_API/$PROJECT_ID/share-networks"
curl -v -s -X POST $OS_NAS_API/$PROJECT_ID/share-networks --header "X-Auth-Token: $OS_TOKEN" 'Content-Type: application/json' \
--data-raw '
{
	"share_network": {
		"name": "kt-dx-m1-zone-subnet-cug-d7euu0j1vojfopcuur9g",
		"neutron_net_id": "d30cb659-3b23-4d3a-8e44-6e2ce7ab20ac",		
	}
}' ; echo 
echo -e "\n"

# String 입력시 양쪽에 [] 기호는 없이 입력해야함.
