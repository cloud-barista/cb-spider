export CONN_CONFIG=aws-ohio-config
export IMAGE_NAME=ami-090717c950a5c34d3
export SPEC_NAME=t3.micro

./parallel_test.sh $1
