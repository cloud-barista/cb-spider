rm -rf /tmp/AwsDriver.so
go build -buildmode=plugin AwsDriver.go
chmod +x AwsDriver.so
mv ./AwsDriver.so /tmp
