#!/bin/bash
# usage: ./one_csp_snapshot.sh <csp>
#   (re)try MyImage and NLB only; one_csp_run.sh already includes both
#   - existing MyImage/NLB are skipped

./03.myimage-01.sh $1
./04.nlb-01.sh $1
