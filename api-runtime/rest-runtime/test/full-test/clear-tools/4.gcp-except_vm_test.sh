export CONN_CONFIG=gcp-ohio-config
#export IMAGE_NAME=ubuntu-minimal-1804-bionic-v20191024
#export IMAGE_NAME=projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024
export IMAGE_NAME=https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024
export SPEC_NAME=f1-micro

./except_vm_delete.sh
