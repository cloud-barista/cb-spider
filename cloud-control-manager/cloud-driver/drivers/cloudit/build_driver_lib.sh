source ../../../../setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=cloudit-driver-v1.0

rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
go build -buildmode=plugin -o ${DRIVERFILENAME}.so ClouditDriver-lib.go
chmod +x ${DRIVERFILENAME}.so
mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH
