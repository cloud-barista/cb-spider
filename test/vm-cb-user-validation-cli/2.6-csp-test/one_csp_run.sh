#!/bin/bash

./00.prepare-00.sh $1
./01.loop-vm-ssh-case-01.sh $1 $2 
./02.disk-01.sh $1
