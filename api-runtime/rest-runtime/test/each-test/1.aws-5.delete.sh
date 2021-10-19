export CONN_CONFIG=aws-ohio-config
export IMAGE_NAME=ami-0bbe28eb2173f6167
export SPEC_NAME=t3.micro

./vm-delete-test.sh
./keypair-delete-test.sh
./securitygroup-delete-test.sh
./vpc-delete-test.sh
