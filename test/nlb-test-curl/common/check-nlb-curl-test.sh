NLB_ADDR=$1
NLB_PORT=$2

for (( num=1; num <= 20; num++ ))
do
	curl -s $NLB_ADDR:$NLB_PORT |grep title
	#sleep 1
done
