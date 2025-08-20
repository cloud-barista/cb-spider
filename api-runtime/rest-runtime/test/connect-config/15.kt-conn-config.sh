RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"kt-driver01","ProviderName":"KT", "DriverLibFileName":"kt-driver-v1.0.so"}'

 # for Cloud Credential Info
 # $$$ Need to append '/v3/' to identity_endpoint URL 
 # $$$ For 'V3' verson auth., identity_endpoint, username, password and domain_name are required basically.
 # $$$ And, need 'project_id' for the token role
 # You can get the prject id on 'Servers' > 'Token' > 'Token' menu on KT Cloud Portal
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{
    "CredentialName":"kt-credential01",
    "ProviderName":"KT",
    "KeyValueInfoList": [
        {"Key":"IdentityEndpoint", "Value":"https://api.ucloudbiz.olleh.com/d1/identity/v3/"},
        {"Key":"Username", "Value":"~~~@~~~.com"},
        {"Key":"Password", "Value":"XXXXXXXXXX"},
        {"Key":"DomainName", "Value":"default"},
        {"Key":"ProjectID", "Value":"XXXXXXXXXX"}
]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"kt-DX-M1-zone","ProviderName":"KT","KeyValueInfoList": [{"Key":"Region", "Value":"KR1"}, {"Key":"Zone", "Value":"DX-M1"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"kt-mokdong1-config","ProviderName":"KT", "DriverName":"kt-driver01", "CredentialName":"kt-credential01", "RegionName":"kt-DX-M1-zone"}'
