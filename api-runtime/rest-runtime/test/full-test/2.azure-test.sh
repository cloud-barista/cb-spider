export CONN_CONFIG=azure-northeu-config
# Now, latest is not supported like 'GetImage("Canonical:UbuntuServer:18.04-LTS:latest")' // export IMAGE_NAME=Canonical:UbuntuServer:18.04-LTS:latest
export IMAGE_NAME=Canonical:UbuntuServer:18.04-LTS:18.04.202109180
export SPEC_NAME=Standard_B1ls

./full_test.sh
