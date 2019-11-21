source ../setup.env

curl -sX POST http://$RESTSERVER:1024/vm?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{ 
	"VMName": "vm-powerkim01", 
	"ImageId": "a846af3b-5d80-4182-b38e-5501ad9f78f4",
	"VirtualNetworkId": "10.0.8.0",
	"SecurityGroupIds": ["064616d0-74fe-4840-8a42-8af4ce24e96a"], 
	"VMSpecId": "1c38e438-ede9-4df5-8775-2ce791698924",
	"VMUserId": "root", 
	"VMUserPasswd": "etriETRI!@"
}' |json_pp
