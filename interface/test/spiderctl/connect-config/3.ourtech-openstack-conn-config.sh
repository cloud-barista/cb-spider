 # for Cloud Driver Info
$CBSPIDER_ROOT/interface/spctl driver create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{
      "DriverName":"openstack-driver01",
      "ProviderName":"OPENSTACK", 
      "DriverLibFileName":"openstack-driver-v1.0.so"
    }'

 # for Cloud Credential Info
$CBSPIDER_ROOT/interface/spctl credential create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{
      "CredentialName":"openstack-credential01",
      "ProviderName":"OPENSTACK",
      "KeyValueInfoList": [
        {"Key":"IdentityEndpoint", "Value":"http://192.168.201.208:5000/v3"},
        {"Key":"Username", "Value":"demo"},
        {"Key":"Password", "Value":"openstack"},
        {"Key":"DomainName", "Value":"Default"},
        {"Key":"ProjectID", "Value":"b31474c562184bcbaf3496e08f5a6a4c"}
      ]
    }'

 # for Cloud Region Info
$CBSPIDER_ROOT/interface/spctl region create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{
      "RegionName":"openstack-region01",
      "ProviderName":"OPENSTACK",
      "KeyValueInfoList": [
        {"Key":"Region", "Value":"RegionOne"}
      ]
    }' 

 # for Cloud Connection Config Info
$CBSPIDER_ROOT/interface/spctl connection create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{
      "ConfigName":"openstack-config01",
      "ProviderName":"OPENSTACK", 
      "DriverName":"openstack-driver01", 
      "CredentialName":"openstack-credential01", 
      "RegionName":"openstack-region01"
    }'
