export CONN_CONFIG=mock-config01
export IMAGE_NAME=mock-vmimage-01
export SPEC_NAME=mock-vmspec-01

# API BasicAuth Header
ApiUsername=default
ApiPassword=default
export AUTH="Authorization: Basic $(echo -n $ApiUsername:$ApiPassword | base64)"

./mock-auth_full_test.sh
