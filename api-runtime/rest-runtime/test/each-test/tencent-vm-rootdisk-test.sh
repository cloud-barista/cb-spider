
echo "####################################################################"
echo "## VM RootDisk Type and Resize Test Scripts for CB-Spider - 2022.02.11."
echo "####################################################################"

#### RootDiskType / RootDiskSize
#### "" / ""
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { 
			"Name": "vm-01", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "vpc-01", 
			"SubnetName": "subnet-01", 
			"SecurityGroupNames": [ "sg-01" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "keypair-01"
		} 
	}' |json_pp

#### RootDiskType / RootDiskSize
#### default / default
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Name": "vm-01-default-default",
                        "ImageName": "'${IMAGE_NAME}'",
                        "VPCName": "vpc-01",
                        "SubnetName": "subnet-01",
                        "SecurityGroupNames": [ "sg-01" ],
                        "VMSpecName": "'${SPEC_NAME}'",
                        "KeyPairName": "keypair-01",
			"RootDiskType": "default",
			"RootDiskSize": "default"
                }
        }' |json_pp

#### RootDiskType / RootDiskSize
#### CLOUD_PREMIUM / ""
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": { 
			"Name": "vm-01-CLOUD_PREMIUM-default",
			"ImageName": "'${IMAGE_NAME}'",
			"VPCName": "vpc-01",
			"SubnetName": "subnet-01",
			"SecurityGroupNames": [ "sg-01" ],
			"VMSpecName": "'${SPEC_NAME}'",
			"RootDiskType": "CLOUD_PREMIUM",
			"KeyPairName": "keypair-01"
		}
        }' |json_pp

#### RootDiskType / RootDiskSize
#### default / 7
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Name": "vm-01-default-7GB",
                        "ImageName": "'${IMAGE_NAME}'",
                        "VPCName": "vpc-01",
                        "SubnetName": "subnet-01",
                        "SecurityGroupNames": [ "sg-01" ],
                        "VMSpecName": "'${SPEC_NAME}'",
                        "RootDiskType": "default",
                        "RootDiskSize": "7",
                        "KeyPairName": "keypair-01"
                }
        }' |json_pp

#### RootDiskType / RootDiskSize
#### default / 60
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Name": "vm-01-default-60GB",
                        "ImageName": "'${IMAGE_NAME}'",
                        "VPCName": "vpc-01",
                        "SubnetName": "subnet-01",
                        "SecurityGroupNames": [ "sg-01" ],
                        "VMSpecName": "'${SPEC_NAME}'",
                        "RootDiskType": "default",
                        "RootDiskSize": "60",
                        "KeyPairName": "keypair-01"
                }
        }' |json_pp


#### RootDiskType / RootDiskSize
#### TYPE1 / default
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Name": "vm-01-type1-default",
                        "ImageName": "'${IMAGE_NAME}'",
                        "VPCName": "vpc-01",
                        "SubnetName": "subnet-01",
                        "SecurityGroupNames": [ "sg-01" ],
                        "VMSpecName": "'${SPEC_NAME}'",
                        "KeyPairName": "keypair-01",
                        "RootDiskType": "TYPE1",
                        "RootDiskSize": "default"
                }
        }' |json_pp


#### RootDiskType / RootDiskSize
#### CLOUD_SSD / 54
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Name": "vm-01-CLOUD_SSD-54GB",
                        "ImageName": "'${IMAGE_NAME}'",
                        "VPCName": "vpc-01",
                        "SubnetName": "subnet-01",
                        "SecurityGroupNames": [ "sg-01" ],
                        "VMSpecName": "'${SPEC_NAME}'",
                        "KeyPairName": "keypair-01",
                        "RootDiskType": "CLOUD_SSD",
                        "RootDiskSize": "54"
                }
        }' |json_pp

