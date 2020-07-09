 # for Cloud Driver Info
$CBSPIDER_ROOT/interface/cbadm driver create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -f ./driver.yaml

 # for Cloud Credential Info
$CBSPIDER_ROOT/interface/cbadm credential create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -f ./cred.yaml

 # for Cloud Region Info
$CBSPIDER_ROOT/interface/cbadm region create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -f ./region.yaml

 # for Cloud Connection Config Info
$CBSPIDER_ROOT/interface/cbadm connect-infos create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -f ./connection-01.yaml

 # Get Cloud Connection Config Info
$CBSPIDER_ROOT/interface/cbadm connect-infos get --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -n openstack-config01

 # List Cloud Connection Config Info
$CBSPIDER_ROOT/interface/cbadm connect-infos list --config $CBSPIDER_ROOT/interface/grpc_conf.yaml