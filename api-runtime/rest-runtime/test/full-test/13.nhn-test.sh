export CONN_CONFIG=nhn-korea-pangyo1-config
#export CONN_CONFIG=nhn-korea-pyeongchon1-config

export IMAGE_NAME=1cc42a4d-53da-4da9-a464-5241839b68c5
# Image Guest OS : Ubuntu Server 24.04.5 LTS (2026.11.19) 기준

export SPEC_NAME=u2.c2m4
# VM Spec : vCPU: 2, Mem: 4GB
# Not need to specify Disk Size incase of u2.~~~ type of VMSpec

#export SPEC_NAME=c2.c2m2
#export SPEC_NAME=m2.c4m8
# VM Spec : vCPU: 4, Mem: 8GB
# Need to specify Disk Type and Disk Size

./nhn-full_test.sh
