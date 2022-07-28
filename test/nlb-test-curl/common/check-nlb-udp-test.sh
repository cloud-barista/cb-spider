NLB_ADDR=$1
NLB_PORT=$2

for (( num=1; num <= 20; num++ ))
do
	ncat -w3 -zu $NLB_ADDR $NLB_PORT
	#sleep 1
done
