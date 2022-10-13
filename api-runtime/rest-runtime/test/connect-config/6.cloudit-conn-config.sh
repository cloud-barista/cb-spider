RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"cloudit-driver01","ProviderName":"CLOUDIT", "DriverLibFileName":"cloudit-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{ "CredentialName":"cloudit-credential01", "ProviderName":"CLOUDIT", "KeyValueInfoList": [{"Key":"IdentityEndpoint", "Value":"http://xxx.xxx.co.kr:9090"}, {"Key":"AuthToken", "Value":"xxxx"}, {"Key":"Username", "Value":"xxxx"}, {"Key":"Password", "Value":"xxxx"}, {"Key":"TenantId", "Value":"tnt0009"}, {"Key":"ClusterId", "Value":"CL"}]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"cloudit-region01","ProviderName":"CLOUDIT", "KeyValueInfoList": [{"Key":"Region", "Value":"default"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"cloudit-config01","ProviderName":"CLOUDIT", "DriverName":"cloudit-driver01", "CredentialName":"cloudit-credential01", "RegionName":"cloudit-region01"}'
