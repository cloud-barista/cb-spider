RESTSERVER=localhost

 # Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"nhn-driver01","ProviderName":"NHN", "DriverLibFileName":"nhn-driver-v1.0.so"}'

 # Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{
    "CredentialName":"nhn-credential01",
    "ProviderName":"NHN",
    "KeyValueInfoList": [
        {"Key":"IdentityEndpoint", "Value":"https://api-identity-infrastructure.nhncloudservice.com"},
        {"Key":"Username", "Value":"XXXXX@XXXXXXXXXXXXXXXX"},
        {"Key":"Password", "Value":"XXXXXXXXXXXXXXXXXX"},
        {"Key":"DomainName", "Value":"default"},
        {"Key":"TenantId", "Value":"XXXXXXXXXXXXXXXXX"}
]}'
    # TenantId : NHN Cloud Console > Instance > 관리 > API 엔드포인트 설정 > TenantID를 입력해야함.

 # Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhn-korea-pangyo1","ProviderName":"NHN","KeyValueInfoList": [{"Key":"Region", "Value":"KR1"}, {"Key":"Zone", "Value":"kr-pub-a"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhn-korea-pangyo2","ProviderName":"NHN","KeyValueInfoList": [{"Key":"Region", "Value":"KR1"}, {"Key":"Zone", "Value":"kr-pub-b"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhn-korea-pyeongchon1","ProviderName":"NHN","KeyValueInfoList": [{"Key":"Region", "Value":"KR2"}, {"Key":"Zone", "Value":"kr2-pub-a"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhn-korea-pyeongchon2","ProviderName":"NHN","KeyValueInfoList": [{"Key":"Region", "Value":"KR2"}, {"Key":"Zone", "Value":"kr2-pub-b"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhn-japan-tokyo1","ProviderName":"NHN","KeyValueInfoList": [{"Key":"Region", "Value":"JP1"}, {"Key":"Zone", "Value":"jp-pub-a"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhn-japan-tokyo2","ProviderName":"NHN","KeyValueInfoList": [{"Key":"Region", "Value":"JP1"}, {"Key":"Zone", "Value":"jp-pub-b"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"nhn-usa-california1","ProviderName":"NHN","KeyValueInfoList": [{"Key":"Region", "Value":"US1"}, {"Key":"Zone", "Value":"us-pub-a"}]}'

 # Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhn-korea-pangyo1-config","ProviderName":"NHN", "DriverName":"nhn-driver01", "CredentialName":"nhn-credential01", "RegionName":"nhn-korea-pangyo1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhn-korea-pangyo2-config","ProviderName":"NHN", "DriverName":"nhn-driver01", "CredentialName":"nhn-credential01", "RegionName":"nhn-korea-pangyo2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhn-korea-pyeongchon1-config","ProviderName":"NHN", "DriverName":"nhn-driver01", "CredentialName":"nhn-credential01", "RegionName":"nhn-korea-pyeongchon1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhn-korea-pyeongchon2-config","ProviderName":"NHN", "DriverName":"nhn-driver01", "CredentialName":"nhn-credential01", "RegionName":"nhn-korea-pyeongchon2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhn-japan-tokyo1-config","ProviderName":"NHN", "DriverName":"nhn-driver01", "CredentialName":"nhn-credential01", "RegionName":"nhn-japan-tokyo1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhn-japan-tokyo2-config","ProviderName":"NHN", "DriverName":"nhn-driver01", "CredentialName":"nhn-credential01", "RegionName":"nhn-japan-tokyo2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"nhn-usa-california1-config","ProviderName":"NHN", "DriverName":"nhn-driver01", "CredentialName":"nhn-credential01", "RegionName":"nhn-usa-california1"}'
