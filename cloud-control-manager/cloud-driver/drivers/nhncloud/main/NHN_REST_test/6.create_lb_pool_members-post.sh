#!/bin/bash
source ./1.export.env

# XXXXX : Pool ID
curl -v -s -X POST 'https://kr1-api-network.infrastructure.cloud.toast.com/v2.0/lbaas/pools/XXXXX/members' -H "X-Auth-Token: $OS_TOKEN" --header 'Content-Type: application/json' \
--data-raw '
{
  "poolId": "XXXXX", 
  "member": {
    "weight": 1,
    "admin_state_up": true,
    "subnet_id": "XXXXX",
    "address": "192.168.0.45",
    "protocol_port": 8080
  }
}' ; echo

# poolId <= 대소문자 주의!!
# 최종 결론 : poolId는 불필요함. NHN Cloud 매뉴얼이 잘못됨.
