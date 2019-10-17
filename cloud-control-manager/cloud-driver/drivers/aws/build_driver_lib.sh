DRIVERLIB_PATH=$CBSTORE_ROOT/cloud-driver/libs
DRIVERFILENAME=aws-driver-v1.0

rm -rf $DRIVERLIB_PATH/$DRIVERFILENAME.so
go build -buildmode=plugin -o ${DRIVERFILENAME}.so AwsDriver-lib.go
chmod +x $DRIVERFILENAME.so
mv ./$DRIVERFILENAME.so $DRIVERLIB_PATH
