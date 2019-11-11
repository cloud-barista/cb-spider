#netstat -ap tcp |grep 1024

STR=`netstat -ap tcp |grep 1024 |awk '{print $7}'`

if [ -z ${STR} ] ; then
	echo CB-Spider Process does not exist!!
else
	echo STR: $STR
	NUM=`expr index "'${STR}'" "/"`
	echo NUM: $NUM
	NUM=`expr $NUM - 2`
	echo NUM-len_space: $NUM

	PID=`expr substr $STR 1 $NUM`
fi

#kill -9 $PID
