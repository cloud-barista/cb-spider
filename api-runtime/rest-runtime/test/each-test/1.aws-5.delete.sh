export CONN_CONFIG=aws-ohio-config
export IMAGE_NAME=ami-090717c950a5c34d3
export SPEC_NAME=t3.micro

./vm-delete-test.sh
./keypair-delete-test.sh
./securitygroup-delete-test.sh
./vpc-delete-test.sh
