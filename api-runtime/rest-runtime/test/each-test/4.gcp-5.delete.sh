export CONN_CONFIG=gcp-ohio-config
export IMAGE_NAME=ubuntu-1804-bionic-v20191002
export SPEC_NAME=f1-micro

./vm-delete-test.sh
./keypair-delete-test.sh
./securitygroup-delete-test.sh
./vpc-delete-test.sh
