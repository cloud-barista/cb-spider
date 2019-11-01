RESTSERVER=localhost

KEY=`curl -X POST http://$RESTSERVER:1024/keypair?connection_name=aws-ohio-config -H 'Content-Type: application/json' -d '{ "Name": "mcb-keypair-powerkim" }' |json_pp | grep PrivateKey |sed 's/"PrivateKey" : "//g' |sed 's/-----",/-----/g' |sed 's/-----"/-----/g'`
echo -e ${KEY}
echo -e ${KEY} > ./aws-ohio.key
chmod 600 ./aws-ohio.key

KEY=`curl -X POST http://$RESTSERVER:1024/keypair?connection_name=aws-oregon-config -H 'Content-Type: application/json' -d '{ "Name": "mcb-keypair-powerkim" }' |json_pp | grep PrivateKey |sed 's/"PrivateKey" : "//g' |sed 's/-----",/-----/g' |sed 's/-----"/-----/g'`
echo -e ${KEY}
echo -e ${KEY} > ./aws-oregon.key
chmod 600 ./aws-oregon.key


KEY=`curl -X POST http://$RESTSERVER:1024/keypair?connection_name=aws-singapore-config -H 'Content-Type: application/json' -d '{ "Name": "mcb-keypair-powerkim" }' |json_pp | grep PrivateKey |sed 's/"PrivateKey" : "//g' |sed 's/-----",/-----/g' |sed 's/-----"/-----/g'`
echo -e ${KEY}
echo -e ${KEY} > ./aws-singapore.key
chmod 600 ./aws-singapore.key


KEY=`curl -X POST http://$RESTSERVER:1024/keypair?connection_name=aws-paris-config -H 'Content-Type: application/json' -d '{ "Name": "mcb-keypair-powerkim" }' |json_pp | grep PrivateKey |sed 's/"PrivateKey" : "//g' |sed 's/-----",/-----/g' |sed 's/-----"/-----/g'`
echo -e ${KEY}
echo -e ${KEY} > ./aws-paris.key
chmod 600 ./aws-paris.key


KEY=`curl -X POST http://$RESTSERVER:1024/keypair?connection_name=aws-saopaulo-config -H 'Content-Type: application/json' -d '{ "Name": "mcb-keypair-powerkim" }' |json_pp | grep PrivateKey |sed 's/"PrivateKey" : "//g' |sed 's/-----",/-----/g' |sed 's/-----"/-----/g'`
echo -e ${KEY}
echo -e ${KEY} > ./aws-saopaulo.key
chmod 600 ./aws-saopaulo.key

KEY=`curl -X POST http://$RESTSERVER:1024/keypair?connection_name=aws-tokyo-config -H 'Content-Type: application/json' -d '{ "Name": "mcb-keypair-powerkim" }' |json_pp | grep PrivateKey |sed 's/"PrivateKey" : "//g' |sed 's/-----",/-----/g' |sed 's/-----"/-----/g'`
echo -e ${KEY}
echo -e ${KEY} > ./aws-tokyo.key
chmod 600 ./aws-tokyo.key
