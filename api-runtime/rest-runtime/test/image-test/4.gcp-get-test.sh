export CONN_CONFIG=gcp-ohio-config
#export IMAGE_NAME=ubuntu-minimal-1804-bionic-v20191024
#export IMAGE_NAME=projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024
#export IMAGE_NAME=https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024
export IMAGE_NAME=https%3A%2F%2Fwww.googleapis.com%2Fcompute%2Fv1%2Fprojects%2Fubuntu-os-cloud%2Fglobal%2Fimages%2Fubuntu-minimal-1804-bionic-v20191024

./image-get-test.sh
