RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/driver -H 'Content-Type: application/json' -d '{"DriverName":"aws-driver01","ProviderName":"AWS", "DriverLibFileName":"aws-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/credential -H 'Content-Type: application/json' -d '{"CredentialName":"aws-credential01","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"ClientId", "Value":"생성된 API용 Access key ID 값을 입력함"}, {"Key":"ClientSecret", "Value":"생성된 API용 Secret access key 값을 입력 함."}]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-region01","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-northeast-2"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-config01","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-region01"}'
