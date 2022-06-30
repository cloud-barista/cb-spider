./0.mock-prepare-nlb-test.sh

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./1.mock-create-nlb-test.sh

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./2.mock-list-get-nlb-test.sh

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./3.mock-get-owner-vpc-nlb-test.sh

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./4.mock-addvm-removevm-nlb-test.sh

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./5.mock-change-nlb-test.sh

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

./6.mock-gethealth-nlb-test.sh
