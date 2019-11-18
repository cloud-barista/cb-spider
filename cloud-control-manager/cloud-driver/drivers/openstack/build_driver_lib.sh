source ../../../../setup.env

DRIVERLIB_PATH=$CBSPIDER_ROOT/cloud-driver-libs
DRIVERFILENAME=openstack-driver-v1.0

rm -rf $DRIVERLIB_PATH/${DRIVERFILENAME}.so
go build -buildmode=plugin -o ${DRIVERFILENAME}.so OpenStackDriver-lib.go
chmod +x ${DRIVERFILENAME}.so
mv ./${DRIVERFILENAME}.so $DRIVERLIB_PATH
