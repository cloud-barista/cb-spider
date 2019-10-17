RESTSERVER=node12

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/driver -H 'Content-Type: application/json' -d '{"DriverName":"azure-driver01","ProviderName":"AZURE", "DriverLibFileName":"azure-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/credential -H 'Content-Type: application/json' -d '{"CredentialName":"azure-credential01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"ClientId", "Value":"xxx"}, {"Key":"ClientSecret", "Value":"xxx"}, {"Key":"TenantId", "Value":"xxx"}, {"Key":"SubscriptionId", "Value":"xxx"}]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/region -H 'Content-Type: application/json' -d '{"RegionName":"azure-region01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"location", "Value":"koreacentral"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"azure-config01","ProviderName":"AZURE", "DriverName":"azure-driver01", "CredentialName":"azure-credential01", "RegionName":"azure-region01"}'
