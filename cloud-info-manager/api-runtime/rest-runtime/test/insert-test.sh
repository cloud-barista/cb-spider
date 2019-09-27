RESTSERVER=node12

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/driver -H 'Content-Type: application/json' -d '{"DriverName":"aws-driver01","ProviderName":"AWS", "DriverLibFileName":"aws-driver-v1.0.so"}'
curl -X POST http://$RESTSERVER:1024/driver -H 'Content-Type: application/json' -d '{"DriverName":"azure-driver01","ProviderName":"AZURE", "DriverLibFileName":"azure-driver-v1.0.so"}'
curl -X POST http://$RESTSERVER:1024/driver -H 'Content-Type: application/json' -d '{"DriverName":"k8s-driver-V0.5","ProviderName":"K8S", "DriverLibFileName":"k8s-driver-v0.5.so"}' 

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/credential -H 'Content-Type: application/json' -d '{"CredentialName":"aws-credential01","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"aws_access_key_id", "Value":"value1"}, {"Key":"aws_secret_access_key", "Value":"value2"}]}'
curl -X POST http://$RESTSERVER:1024/credential -H 'Content-Type: application/json' -d '{"CredentialName":"azure-credential01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXX"}, {"Key":"ClientSecret", "Value":"XXX"}, {"Key":"TenantId", "Value":"XXX"}, {"Key":"SubscriptionId", "Value":"XXX"}]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-region01","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"region", "Value":"ap-northeast-2"}]}'
curl -X POST http://$RESTSERVER:1024/region -H 'Content-Type: application/json' -d '{"RegionName":"azure-region01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"location", "Value":"koreacentral"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-config01","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-region01"}'
curl -X POST http://$RESTSERVER:1024/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"azure-config01","ProviderName":"AZURE", "DriverName":"azure-driver01", "CredentialName":"azure-credential01", "RegionName":"azure-region01"}'
