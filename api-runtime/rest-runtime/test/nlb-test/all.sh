./0.prepare-nlb-test.sh $1

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./1.create-nlb-test.sh $1

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./2.list-get-nlb-test.sh $1

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./3.get-owner-vpc-nlb-test.sh $1

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./4.addvm-removevm-nlb-test.sh $1

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./5.change-nlb-test.sh $1

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./6.gethealth-nlb-test.sh $1
