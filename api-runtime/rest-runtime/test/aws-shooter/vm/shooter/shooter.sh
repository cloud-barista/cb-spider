SERVER=18.176.22.52


DT=`date`
DT=`echo $DT |sed 's/ /%20/g'`
HN=`hostname`

curl -X GET http://$SERVER:119/test -H 'Content-Type: application/json' -d '{"Date": "'${DT}'", "HostName": "'${HN}'"}'
