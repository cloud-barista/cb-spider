RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"tencent-driver01","ProviderName":"TENCENT", "DriverLibFileName":"tencent-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"tencent-credential01","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"ClientId", "Value":"AKIDg8JEwNQ12pKaaet8YhbFdlBM2f6mbWaZ"}, {"Key":"ClientSecret", "Value":"2use6MQJFzbM6iJDtcASWNn3FD0Yv42l"}]}'

 # Cloud Region Info for Shooter
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-tokyo","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-tokyo"}, {"Key":"Zone", "Value":"ap-tokyo-2"}]}'

 # Cloud Connection Config Info for Shooter
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-tokyo-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-tokyo"}'
