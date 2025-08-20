echo "curl -v -s -X POST https://api.ucloudbiz.olleh.com/d1/identity/auth/tokens"
curl -v -s -X POST 'https://api.ucloudbiz.olleh.com/d1/identity/auth/tokens' --header 'Content-Type: application/json' \
--data-raw '
{ 
  "auth": {
    "identity": {
      "methods": ["password"],
      "password": {
        "user": {
          "domain": { 
            "id": "default" 
          },
          "name": "~~~~~@~~~~~~~~",
          "password": "XXXXXXXXXX"
        }
      }
    },

    "scope": {
      "project": {
        "domain": {
          "id": "default"
        },
        "name": "~~~~~@~~~~~~~~"
      }
    }
  }
}' ; echo 

echo -e "\n"

# String 입력시 양쪽에 [] 기호는 없이 입력해야함.
