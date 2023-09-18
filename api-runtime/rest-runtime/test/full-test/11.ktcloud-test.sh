# Image Guest OS : Ubuntu Linux 18.04 기준
# VM Spec : vCPU: 4, Mem: 8GB

export CONN_CONFIG=ktcloud-korea-cheonan2-config
export IMAGE_NAME=97ef0091-fdf7-44e9-be79-c99dc9b1a0ad
export SPEC_NAME=d3530ad2-462b-43ad-97d5-e1087b952b7d!87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB

# Seoul2
# export IMAGE_NAME=63de6d04-7f1b-4924-8e95-1acd6581ca0c
# export SPEC_NAME=3842610a-8b04-4796-a7cd-4ee3706a4666

./ktcloud-full_test.sh
