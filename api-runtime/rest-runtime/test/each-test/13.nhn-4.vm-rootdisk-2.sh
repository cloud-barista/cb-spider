export CONN_CONFIG=nhn-korea-pangyo-config
#export CONN_CONFIG=nhn-korea-pyeongchon-config

export IMAGE_NAME=1cc42a4d-53da-4da9-a464-5241839b68c5
# Image Guest OS : Ubuntu Server 24.04.5 LTS (2026.11.19) 기준

# export SPEC_NAME=u2.c2m4
# VM Spec : vCPU: 2, Mem: 4GB, Local Disk : 50GB

#export SPEC_NAME=c2.c2m2
export SPEC_NAME=m2.c4m8
# VM Spec : vCPU: 4, Mem: 8GB, Need to Attach Block Disk

./nhn-vm-rootdisk-test-2.sh
