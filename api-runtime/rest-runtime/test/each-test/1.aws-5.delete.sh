export CONN_CONFIG=aws-ohio-config
export IMAGE_NAME=ami-00978328f54e31526
export SPEC_NAME=t3.micro

./vm-delete-test.sh
./keypair-delete-test.sh
./securitygroup-delete-test.sh
./vpc-delete-test.sh
