# Image Guest OS : Ubuntu Linux 18.04 기준
# VM Spec : vCPU: 2, Mem: 4GB

export CONN_CONFIG=nhncloud-korea-pangyo-config
#export CONN_CONFIG=nhncloud-korea-pyeongchon-config

export IMAGE_NAME=5396655e-166a-4875-80d2-ed8613aa054f
export SPEC_NAME=u2.c2m4

./nhncloud-full_test.sh
