echo "curl -v -s -X POST https://api-identity-infrastructure.nhncloudservice.com/v2.0/tokens"
curl -v -s -X POST 'https://api-identity-infrastructure.nhncloudservice.com/v2.0/tokens' --header 'Content-Type: application/json' \
--data-raw '
{ 
  "auth": {
    "tenantId": "XXXXXXXXXX",
    "passwordCredentials": {
      "username": "~~~@~~~.com",
      "password": "XXXXX-API-PaasWord-XXXXX"
      }
    }
  }
}' ; echo 

echo -e "\n"

# IAM 계정 정보를 입력하면 인증 않됨.
# String 입력시 양쪽에 [] 기호는 없이 입력해야함.
