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
    # TenantId : NHN Cloud Console > Instance > 관리 > API 엔드포인트 설정 > TenantID를 입력해야함.

 # Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-korea-pangyo1","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"KR1"}, {"Key":"Zone", "Value":"kr-pub-a"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-korea-pangyo2","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"KR1"}, {"Key":"Zone", "Value":"kr-pub-b"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-korea-pyeongchon1","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"KR2"}, {"Key":"Zone", "Value":"kr2-pub-a"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-korea-pyeongchon2","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"KR2"}, {"Key":"Zone", "Value":"kr2-pub-b"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-japan-tokyo1","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"JP1"}, {"Key":"Zone", "Value":"jp-pub-a"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhncloud-japan-tokyo2","ProviderName":"NHNCLOUD","KeyValueInfoList": [{"Key":"Region", "Value":"JP1"}, {"Key":"Zone", "Value":"jp-pub-b"}]}'

 # Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-korea-pangyo1-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-korea-pangyo1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-korea-pangyo2-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-korea-pangyo2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-korea-pyeongchon1-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-korea-pyeongchon1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-korea-pyeongchon2-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-korea-pyeongchon2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-japan-tokyo1-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-japan-tokyo1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhncloud-japan-tokyo2-config","ProviderName":"NHNCLOUD", "DriverName":"nhncloud-driver01", "CredentialName":"nhncloud-credential01", "RegionName":"nhncloud-japan-tokyo2"}'

