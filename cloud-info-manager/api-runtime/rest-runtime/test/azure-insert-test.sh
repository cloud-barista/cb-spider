RESTSERVER=node12

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/driver -H 'Content-Type: application/json' -d '{"DriverName":"azure-driver01","ProviderName":"AZURE", "DriverLibFileName":"azure-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/credential -H 'Content-Type: application/json' -d '{"CredentialName":"azure-credential01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"ClientId", "Value":"a3315693-a8f4-49a4-a0f5-8eff80b3c242"}, {"Key":"ClientSecret", "Value":"aec8cf6f-6933-4ac3-8aae-c54854a5477a"}, {"Key":"TenantId", "Value":"82a99008-10c9-41bb-ad72-8fa46f6fe1cb"}, {"Key":"SubscriptionId", "Value":"f1548292-2be3-4acd-84a4-6df079160846"}]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/region -H 'Content-Type: application/json' -d '{"RegionName":"azure-region01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"location", "Value":"koreacentral"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"azure-config01","ProviderName":"AZURE", "DriverName":"azure-driver01", "CredentialName":"azure-credential01", "RegionName":"azure-region01"}'
