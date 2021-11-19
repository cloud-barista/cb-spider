RESTSERVER=localhost

curl -X POST "http://$RESTSERVER:1024/spider/driver" -H 'Content-Type: application/json' -d '{"DriverName":"gcp-driver01","ProviderName":"GCP", "DriverLibFileName":"gcp-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST "http://$RESTSERVER:1024/spider/credential" -H 'Content-Type: application/json' -d '{"CredentialName":"gcp-credential01","ProviderName":"GCP", "KeyValueInfoList": [{"Key":"PrivateKey", "Value":"-----BEGIN PRIVATE KEY-----\nXXXX\n-----END PRIVATE KEY-----\n"},{"Key":"ProjectID", "Value":"powerkimhub"}, {"Key":"ClientEmail", "Value":"xxxx@xxxx.iam.gserviceaccount.com"}]}'

 # for Cloud Region Info
curl -X POST "http://$RESTSERVER:1024/spider/region" -H 'Content-Type: application/json' -d '{"RegionName":"gcp-ohio-region","ProviderName":"GCP", "KeyValueInfoList": [{"Key":"Region", "Value":"us-central1"},{"Key":"Zone", "Value":"us-central1-a"}]}'

 # for Cloud Connection Config Info
curl -X POST "http://$RESTSERVER:1024/spider/connectionconfig" -H 'Content-Type: application/json' -d '{"ConfigName":"gcp-iowa-config","ProviderName":"GCP", "DriverName":"gcp-driver01", "CredentialName":"gcp-credential01", "RegionName":"gcp-ohio-region"}'

