#!/bin/bash
# usage: ./one_csp_run.sh <csp> [number-of-VMs, default: 1]
#   VPC/Subnet, SecurityGroup, KeyPair => VMs => Disk => Object Storage(S3)
#   => MyImage (waits until VM-1 is Running) => NLB => Cluster/NodeGroup

VM_COUNT=${2:-1}

./00.prepare-00.sh $1
./01.loop-vm-ssh-case-01.sh $1 $VM_COUNT
./02.disk-01.sh $1
./05.s3-01.sh $1
./03.myimage-01.sh $1
./04.nlb-01.sh $1
./06.cluster-01.sh $1
