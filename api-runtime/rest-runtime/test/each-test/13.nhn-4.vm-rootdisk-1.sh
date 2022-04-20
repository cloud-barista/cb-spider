export CONN_CONFIG=nhncloud-korea-pangyo-config
#export CONN_CONFIG=nhncloud-korea-pyeongchon-config

export IMAGE_NAME=5396655e-166a-4875-80d2-ed8613aa054f
# Image Guest OS : Ubuntu Linux 18.04 기준

export SPEC_NAME=u2.c2m4
# VM Spec : vCPU: 2, Mem: 4GB, Local Disk : 50GB

#export SPEC_NAME=c2.c2m2
#export SPEC_NAME=m2.c4m8
# VM Spec : vCPU: 4, Mem: 8GB, Need to Attach Block Disk

./nhn-vm-rootdisk-test-1.sh
