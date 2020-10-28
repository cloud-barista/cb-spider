RESTSERVER=localhost

 # Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"ncp-driver01","ProviderName":"NCP", "DriverLibFileName":"ncp-driver-v1.0.so"}'

 # Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"ncp-credential01","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"AccessKeyID", "Value":"XXXXXXXXXXXXXXXXXXX"}, {"Key":"SecretKey", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXX"}]}'

 # Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-korea","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"region", "Value":"KR"}]}'

 # Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-config01","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-korea"}'

