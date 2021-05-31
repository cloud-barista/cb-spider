export CONN_CONFIG=openstack-config01
export IMAGE_NAME=cirros-0.5.1
export SPEC_NAME=m1.tiny

export IPv4_CIDR=10.0.0.0/20
export IPv4_CIDR_SUBNET1=10.0.1.0/24
export IPv4_CIDR_SUBNET2=10.0.2.0/24

./vpc-test.sh
