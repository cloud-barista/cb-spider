export CONN_CONFIG=nhncloud-korea-pangyo1-config
#export CONN_CONFIG=nhncloud-korea-pyeongchon1-config

export IMAGE_NAME=5396655e-166a-4875-80d2-ed8613aa054f
# Image Guest OS : Ubuntu Linux 18.04 기준

export SPEC_NAME=u2.c2m4
# VM Spec : vCPU: 2, Mem: 4GB
# Not need to specify Disk Size incase of u2.~~~ type of VMSpec

#export SPEC_NAME=c2.c2m2
#export SPEC_NAME=m2.c4m8
# VM Spec : vCPU: 4, Mem: 8GB
# Need to specify Disk Type and Disk Size

./nhncloud-full_test.sh
