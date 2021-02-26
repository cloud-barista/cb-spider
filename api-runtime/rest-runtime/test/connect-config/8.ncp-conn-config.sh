RESTSERVER=localhost

 # Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"ncp-driver01","ProviderName":"NCP", "DriverLibFileName":"ncp-driver-v1.0.so"}'

 # Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"ncp-credential01","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXXXXXXXXXXXXXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXX"}]}'

 # Cloud Region & Zone Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-korea1","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"KR"}, {"Key":"Zone", "Value":"KR-1"}]}'
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-korea2","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"KR"}, {"Key":"Zone", "Value":"KR-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-hongkong","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"HK"}, {"Key":"Zone", "Value":"HK-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-singapore","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"SGN"}, {"Key":"Zone", "Value":"SGN-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-japan","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"JPN"}, {"Key":"Zone", "Value":"JPN-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-us-western","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"USWN"}, {"Key":"Zone", "Value":"USWN-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-germany","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"DEN"}, {"Key":"Zone", "Value":"DEN-1"}]}'

 # Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-korea1-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-korea1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-korea2-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-korea2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-hongkong-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-hongkong"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-singapore-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-singapore"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-japan-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-japan"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-us-western-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-us-western"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-germany-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-germany"}'