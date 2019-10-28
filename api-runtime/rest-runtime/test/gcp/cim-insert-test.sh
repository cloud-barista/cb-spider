RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST "http://$RESTSERVER:1024/driver" -H 'Content-Type: application/json' -d '{"DriverName":"gcp-driver01","ProviderName":"GCP", "DriverLibFileName":"gcp-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST "http://$RESTSERVER:1024/credential" -H 'Content-Type: application/json' -d '{"CredentialName":"gcp-credential01","ProviderName":"GCP", "KeyValueInfoList": [{"Key":"ClientSecret", "Value":"/Users/thaeao/Keystore/mcloud-barista-251102-1569f817b
d23.json"},{"Key":"ProjectID", "Value":"mcloud-barista-251102"}, {"Key":"ClientEmail", "Value":"675581125193-compute@developer.gserviceaccount.com"}]}'

 # for Cloud Region Info
curl -X POST "http://$RESTSERVER:1024/region" -H 'Content-Type: application/json' -d '{"RegionName":"gcp-region01","ProviderName":"GCP", "KeyValueInfoList": [{"Key":"Region", "Value":"asia-northeast1"},{"Key":"Zone", "Value":"asia-northeast1-b"}]}'

 # for Cloud Connection Config Info
curl -X POST "http://$RESTSERVER:1024/connectionconfig" -H 'Content-Type: application/json' -d '{"ConfigName":"gcp-config01","ProviderName":"GCP", "DriverName":"gcp-driver01", "CredentialName":"gcp-credential01", "RegionName":"gcp-region01"}'
