export CONN_CONFIG=gcp-iowa-config
#export IMAGE_NAME=ubuntu-1804-bionic-v20191002
export IMAGE_NAME=https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024
export SPEC_NAME=f1-micro

./securitygroup-icmp-test.sh
