RESTSERVER=localhost
#RESTSERVER=powerkimpi

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"azure-driver01","ProviderName":"AZURE", "DriverLibFileName":"azure-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"azure-credential01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXX-XXXX"}, {"Key":"ClientSecret", "Value":"xxxx-xxxx"}, {"Key":"TenantId", "Value":"xxxx-xxxx"}, {"Key":"SubscriptionId", "Value":"xxxx-xxxx"}]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"azure-northeu","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"location", "Value":"northeurope"}, {"Key":"ResourceGroup", "Value":"CB-GROUP-POWERKIM"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"azure-northeu-config","ProviderName":"AZURE", "DriverName":"azure-driver01", "CredentialName":"azure-credential01", "RegionName":"azure-northeu"}'


