export CONN_CONFIG=openstack-config01
export IMAGE_NAME=cirros-0.5.1
export SPEC_NAME=m1.tiny

./vm-delete-test.sh
./keypair-delete-test.sh
./securitygroup-delete-test.sh
./vpc-delete-test.sh
