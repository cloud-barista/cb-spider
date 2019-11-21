RESTSERVER=node12

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/driver -H 'Content-Type: application/json' -d '{"DriverName":"openstack-driver01","ProviderName":"OPENSTACK", "DriverLibFileName":"openstack-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/credential -H 'Content-Type: application/json' -d '{
    "CredentialName":"openstack-credential01",
    "ProviderName":"OPENSTACK",
    "KeyValueInfoList": [
        {"Key":"IdentityEndpoint", "Value":"http://xxx.xxx.xxx.xxx:5000/v3"},
        {"Key":"Username", "Value":"xxx"},
        {"Key":"Password", "Value":"xxx"},
        {"Key":"DomainName", "Value":"default"},
        {"Key":"ProjectID", "Value":"xxx"}
]}'

 # for Cloud Region Info
curl -X POST http://$RESTSERVER:1024/region -H 'Content-Type: application/json' -d '{"RegionName":"openstack-region01","ProviderName":"OPENSTACK","KeyValueInfoList": [{"Key":"Region", "Value":"RegionOne"}]}'

 # for Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"openstack-config01","ProviderName":"OPENSTACK", "DriverName":"openstack-driver01", "CredentialName":"openstack-credential01", "RegionName":"openstack-region01"}'

