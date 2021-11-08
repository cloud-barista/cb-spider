# Image Guest OS : Ubuntu Linux 18.04 LTS Bionic Beaver Minimal Install (amd64) 기준
# VM Spec : vCPU: 2, Mem: 8GB

export CONN_CONFIG=ibmvpc-us-east-1-config
export IMAGE_NAME=r014-dc446598-a1b5-41c3-a1d6-add3afaf264e

# export CONN_CONFIG=ibmvpc-jp-tok-1-config
# export IMAGE_NAME=r022-61fdadec-6b03-4bd2-bfca-62cd16f5673f

# export CONN_CONFIG=ibmvpc-jp-osa-1-config
# export IMAGE_NAME=r034-522c639c-52e1-4cab-8dfb-bc0fb9f6f577

# export CONN_CONFIG=ibmvpc-eu-gb-1-config
# export IMAGE_NAME=r018-1d7417c6-893e-49d4-b14d-9643d6b29812

# export CONN_CONFIG=ibmvpc-eu-de-1-config
# export IMAGE_NAME=r010-1f68eb2d-f35c-4959-8f4b-2b2f9cf78102

# export CONN_CONFIG=ibmvpc-ca-tor-1-config
# export IMAGE_NAME=r038-e92647cf-8be9-438a-b94c-251cc86bc99a

# export CONN_CONFIG=ibmvpc-br-sao-1-config
# export IMAGE_NAME=r026-a8c25ce6-0ca1-43e9-9b41-411c6217b8b8

# export CONN_CONFIG=ibmvpc-us-south-1-config
# export IMAGE_NAME=r006-ed3f775f-ad7e-4e37-ae62-7199b4988b00

# export CONN_CONFIG=ibmvpc-au-syd-1-config
# export IMAGE_NAME=r026-a8c25ce6-0ca1-43e9-9b41-411c6217b8b8

export SPEC_NAME=bx2-2x8

./ibmvpc-full_test.sh
