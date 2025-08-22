# Image Guest OS : Ubuntu Linux 18.04 LTS Bionic Beaver Minimal Install (amd64) 기준
# VM Spec : vCPU: 2, Mem: 8GB

export CONN_CONFIG=ibm-us-east-1-config
export IMAGE_NAME=r014-a044e2f5-dfe1-416c-8990-5dc895352728

# export CONN_CONFIG=ibm-jp-tok-1-config
# export IMAGE_NAME=r022-61fdadec-6b03-4bd2-bfca-62cd16f5673f

# export CONN_CONFIG=ibm-jp-osa-1-config
# export IMAGE_NAME=r034-522c639c-52e1-4cab-8dfb-bc0fb9f6f577

# export CONN_CONFIG=ibm-eu-gb-1-config
# export IMAGE_NAME=r018-1d7417c6-893e-49d4-b14d-9643d6b29812

# export CONN_CONFIG=ibm-eu-de-1-config
# export IMAGE_NAME=r010-1f68eb2d-f35c-4959-8f4b-2b2f9cf78102

# export CONN_CONFIG=ibm-ca-tor-1-config
# export IMAGE_NAME=r038-e92647cf-8be9-438a-b94c-251cc86bc99a

# export CONN_CONFIG=ibm-br-sao-1-config
# export IMAGE_NAME=r042-92d1cd12-f014-4b9a-abf8-c5ca6494a9e5

# export CONN_CONFIG=ibm-us-south-1-config
# export IMAGE_NAME=r006-9de77234-3189-42f8-982d-f2266477cfe0

# export CONN_CONFIG=ibm-au-syd-1-config
# export IMAGE_NAME=r026-a8c25ce6-0ca1-43e9-9b41-411c6217b8b8

export SPEC_NAME=bx2-2x8

./ibm-full_test.sh
