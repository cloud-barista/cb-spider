NLB_ADDR=3.20.19.158

while true
do
	ssh-keygen -f "/home/ubuntu/.ssh/known_hosts" -R $NLB_ADDR > /dev/null
	ssh -i ~/.aws/powerkim-nlb-test.pem -o StrictHostKeyChecking=no $NLB_ADDR hostname 2> /dev/null
done
