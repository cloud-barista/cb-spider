RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"alibaba-driver01","ProviderName":"ALIBABA", "DriverLibFileName":"alibaba-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"alibaba-credential01","ProviderName":"ALIBABA", "KeyValueInfoList": [{"Key":"ClientId", "Value":"xxxxxx"}, {"Key":"ClientSecret", "Value":"xxxxxx"}]}'

 # Cloud Region Info for Shooter
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"alibaba-tokyo","ProviderName":"ALIBABA", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-northeast-1"}]}'

 # Cloud Connection Config Info for Shooter
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"alibaba-tokyo-config","ProviderName":"ALIBABA", "DriverName":"alibaba-driver01", "CredentialName":"alibaba-credential01", "RegionName":"alibaba-tokyo"}'
