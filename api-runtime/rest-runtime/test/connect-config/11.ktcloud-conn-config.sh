RESTSERVER=localhost

# Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"ktcloud-driver01","ProviderName":"KTCLOUD", "DriverLibFileName":"ktcloud-driver-v1.0.so"}'

# Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"ktcloud-credential01","ProviderName":"KTCLOUD", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"}]}'

### Note!!
### In the KT Cloud CSP, it is divided into Zone units without Region units, but the Region information is written for convenience.
### The Zone ID of the following items requires modification after search using 'ListRegionZone()' for each account of KT Cloud Service.

# Cloud Region & Zone Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ktcloud-korea-seoul1","ProviderName":"KTCLOUD", "KeyValueInfoList": [{"Key":"Region", "Value":"KOR-Seoul"}, {"Key":"Zone", "Value":"95e2f517-d64a-4866-8585-5177c256f7c7"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ktcloud-korea-seoul2","ProviderName":"KTCLOUD", "KeyValueInfoList": [{"Key":"Region", "Value":"KOR-Seoul"}, {"Key":"Zone", "Value":"d7d0177e-6cda-404a-a46f-a5b356d2874e"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ktcloud-korea-cheonan1","ProviderName":"KTCLOUD", "KeyValueInfoList": [{"Key":"Region", "Value":"KOR-Central"}, {"Key":"Zone", "Value":"eceb5d65-6571-4696-875f-5a17949f3317"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ktcloud-korea-cheonan2","ProviderName":"KTCLOUD", "KeyValueInfoList": [{"Key":"Region", "Value":"KOR-Central"}, {"Key":"Zone", "Value":"9845bd17-d438-4bde-816d-1b12f37d5080"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ktcloud-korea-kimhae","ProviderName":"KTCLOUD", "KeyValueInfoList": [{"Key":"Region", "Value":"KOR-HA"}, {"Key":"Zone", "Value":"dfd6f03d-dae5-458e-a2ea-cb6a55d0d994"}]}'

# Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ktcloud-korea-seoul1-config","ProviderName":"KTCLOUD", "DriverName":"ktcloud-driver01", "CredentialName":"ktcloud-credential01", "RegionName":"ktcloud-korea-seoul1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ktcloud-korea-seoul2-config","ProviderName":"KTCLOUD", "DriverName":"ktcloud-driver01", "CredentialName":"ktcloud-credential01", "RegionName":"ktcloud-korea-seoul2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ktcloud-korea-cheonan1-config","ProviderName":"KTCLOUD", "DriverName":"ktcloud-driver01", "CredentialName":"ktcloud-credential01", "RegionName":"ktcloud-korea-cheonan1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ktcloud-korea-cheonan2-config","ProviderName":"KTCLOUD", "DriverName":"ktcloud-driver01", "CredentialName":"ktcloud-credential01", "RegionName":"ktcloud-korea-cheonan2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ktcloud-korea-kimhae-config","ProviderName":"KTCLOUD", "DriverName":"ktcloud-driver01", "CredentialName":"ktcloud-credential01", "RegionName":"ktcloud-korea-kimhae"}'
