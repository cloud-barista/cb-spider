RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"openstack-driver01","ProviderName":"OPENSTACK", "DriverLibFileName":"openstack-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{
    "CredentialName":"openstack-credential01",
    "ProviderName":"OPENSTACK",
    "KeyValueInfoList": [
        {"Key":"IdentityEndpoint", "Value":"http://182.252.xxx.xxx:5000/v3"},
        {"Key":"Username", "Value":"etri"},
        {"Key":"Password", "Value":"xxxx"},
        {"Key":"DomainName", "Value":"default"},
        {"Key":"ProjectID", "Value":"xxxx"}
]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"openstack-region01","ProviderName":"OPENSTACK","KeyValueInfoList": [{"Key":"Region", "Value":"RegionOne"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"openstack-config01","ProviderName":"OPENSTACK", "DriverName":"openstack-driver01", "CredentialName":"openstack-credential01", "RegionName":"openstack-region01"}'
