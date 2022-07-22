CSP=$1
NLB_ADDR=$2
NLB_PORT=$3

for (( num=1; num <= 20; num++ ))
do
	ssh-keygen -f "$HOME/.ssh/known_hosts" -R $NLB_ADDR > /dev/null 2> /dev/null
	ssh -i $CSP-keypair-01.pem -o StrictHostKeyChecking=no cb-user@$NLB_ADDR -p $NLB_PORT hostname 2> /dev/null
	#sleep 1
done
