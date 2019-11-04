RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/driver -H 'Content-Type: application/json' -d '{"DriverName":"azure-driver01","ProviderName":"AZURE", "DriverLibFileName":"azure-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/credential -H 'Content-Type: application/json' -d '{"CredentialName":"azure-credential01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"ClientId", "Value":"d783322f-d23d-4acd-b62d-xxxx"}, {"Key":"ClientSecret", "Value":"2d2a43c9-6c0e-49d3-a543-xxxxxxxxx"}, {"Key":"TenantId", "Value":"82a99008-10c9-41bb-ad72-xxxxxxxxxxxxx"}, {"Key":"SubscriptionId", "Value":"f1548292-2be3-4acd-84a4-xxxxxxxxxx"}]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/region -H 'Content-Type: application/json' -d '{"RegionName":"azure-region01","ProviderName":"AZURE", "KeyValueInfoList": [{"Key":"location", "Value":"koreacentral"}, {"Key":"ResourceGroup", "Value":"CB-GROUP"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"azure-config01","ProviderName":"AZURE", "DriverName":"azure-driver01", "CredentialName":"azure-credential01", "RegionName":"azure-region01"}'
