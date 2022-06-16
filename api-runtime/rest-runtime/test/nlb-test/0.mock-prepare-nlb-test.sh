export CONN_CONFIG=mock-config01
export IMAGE_NAME=mock-vmimage-01
export SPEC_NAME=mock-vmspec-01

# Create: VPC/Subnet => SG => Key => VM1, VM2
./prepare-nlb-test.sh
