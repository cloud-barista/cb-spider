RESTSERVER=localhost

 # Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"nhncloud-driver01","ProviderName":"NHNCLOUD", "DriverLibFileName":"nhncloud-driver-v1.0.so"}'

 # Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{
    "CredentialName":"nhncloud-credential01",
    "ProviderName":"NHNCLOUD",
    "KeyValueInfoList": [
        {"Key":"IdentityEndpoint", "Value":"https://api-identity.infrastructure.cloud.toast.com"},
        {"Key":"Username", "Value":"XXXXX@XXXXXXXXXXXXXXXX"},
        {"Key":"Password", "Value":"XXXXXXXXXXXXXXXXXX"},
        {"Key":"DomainName", "Value":"default"},
        {"Key":"TenantId", "Value":"XXXXXXXXXXXXXXXXX"}
]}'

 # Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-korea-pangyo","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"KR1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-korea-pyeongchon","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"KR2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-japan-tokyo","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"JP1"}]}'

 # Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-korea-pangyo-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-korea-pangyo"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-korea-pyeongchon-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-korea-pyeongchon"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-japan-tokyo-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-japan-tokyo"}'
