export CONN_CONFIG=azure-northeu-config
export IMAGE_NAME=Canonical:UbuntuServer:18.04-LTS:latest
export SPEC_NAME=Standard_B1ls

./vm-delete-test.sh
./keypair-delete-test.sh
./securitygroup-delete-test.sh
./vpc-delete-test.sh
