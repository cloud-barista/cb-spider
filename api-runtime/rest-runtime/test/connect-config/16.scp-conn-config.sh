RESTSERVER=localhost

 # for Cloud Driver Info
curl -u admin:XXXXX -sX POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"scp-driver01","ProviderName":"SCP", "DriverLibFileName":"scp-driver-v1.0.so"}'

 # for Cloud Credential Info
 # $$$ SCP Open API uses HMAC-SHA256 request signing with an Access Key / Secret Key pair.
 # $$$ You can get the keys on 'My' > 'My Info' > 'Authentication Key Management' in the SCP Console.
 # $$$ ClientId        : SCP Access Key (SCP_ACCESS_KEY)
 # $$$ ClientSecret    : SCP Secret Key (SCP_SECRET_KEY)
 # $$$ IdentityEndpoint: (Optional) https://identity.{env}.samsungsdscloud.com
 # $$$                   env: s (Samsung) | g (Sovereign) | e (Enterprise), default is 'e'
 # $$$ StsToken        : (Optional) session token for temporary security credentials (assume-role)
curl -u admin:XXXXX -sX POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{
    "CredentialName":"scp-credential01",
    "ProviderName":"SCP",
    "KeyValueInfoList": [
        {"Key":"ClientId", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXXX"},
        {"Key":"ClientSecret", "Value":"XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXXXXXXX"},
        {"Key":"IdentityEndpoint", "Value":"https://identity.e.samsungsdscloud.com"}
]}'

 # for Cloud Region Info
 # $$$ The SCP Open API does not provide any API to list Regions/Zones,
 # $$$ so the supported Regions/Zones depend on the environment of the IdentityEndpoint above.
 # $$$ (Ref: 'Environment and Region List' of https://docs.e.samsungsdscloud.com/apireference/api-common/)
 # $$$   Samsung(s)    : kr-west1 (zones: kr-west1-a, kr-west1-b), kr-east1 (zone: kr-east1-a)
 # $$$   Sovereign(g)  : kr-south1, kr-south2, kr-south3 (zone: {region}-a)
 # $$$   Enterprise(e) : kr-west1 (zones: kr-west1-a, kr-west1-b), kr-east1 (zone: kr-east1-a)
 # $$$ Region : Region code, Zone : Zone code in the Region (e.g. kr-west1-a)
curl -u admin:XXXXX -sX POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"scp-kr-west1","ProviderName":"SCP","KeyValueInfoList": [{"Key":"Region", "Value":"kr-west1"}, {"Key":"Zone", "Value":"kr-west1-b"}]}'

curl -u admin:XXXXX -sX POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"scp-kr-east1","ProviderName":"SCP","KeyValueInfoList": [{"Key":"Region", "Value":"kr-east1"}, {"Key":"Zone", "Value":"kr-east1-a"}]}'
  
 # for Cloud Connection Config Info
curl -u admin:XXXXX -sX POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"scp-kr-west1-config","ProviderName":"SCP", "DriverName":"scp-driver01", "CredentialName":"scp-credential01", "RegionName":"scp-kr-west1"}'

curl -u admin:XXXXX -sX POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"scp-kr-east1-config","ProviderName":"SCP", "DriverName":"scp-driver01", "CredentialName":"scp-credential01", "RegionName":"scp-kr-east1"}'
