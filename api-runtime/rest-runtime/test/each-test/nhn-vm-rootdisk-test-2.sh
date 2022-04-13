echo "#########################################################################"
echo "## VM RootDisk Type and Resize Test Scripts for CB-Spider - 2022.04.12."
echo "#########################################################################"

# $$$ In case of SPEC_NAME=m2.c4m8 $$$

#### RootDiskType / RootDiskSize
#### "" / ""
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { 
			"Name": "nhn-vm-20", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01"
		} 
	}' |json_pp

#### RootDiskType / RootDiskSize
#### default / default
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
    '{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-21", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "default",
			"RootDiskSize": "default"
		}
	}' |json_pp

#### RootDiskType / RootDiskSize
#### standard / ""
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": { 
			"Name": "nhn-vm-22", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_HDD",
			"RootDiskSize": "50"
		}
	}' |json_pp

#### RootDiskType / RootDiskSize
#### General_HDD / 20
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-23", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_HDD",
			"RootDiskSize": "550"
		}
	}' |json_pp

#### RootDiskType / RootDiskSize
#### General_SSD / 50
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-24", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_SSD",
			"RootDiskSize": "77"
		}
	}' |json_pp


#### RootDiskType / RootDiskSize
#### TYPE1 / default
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-25", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "TYPE1",
			"RootDiskSize": "default"
		}
	}' |json_pp

#### RootDiskType / RootDiskSize
#### General_SSD / 5000
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-26", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_SSD",
			"RootDiskSize": "5000"
		}
	}' |json_pp

#### RootDiskType / RootDiskSize
#### General_SSD / default
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-27", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_SSD",
			"RootDiskSize": "default"
		}
	}' |json_pp


#### RootDiskType / RootDiskSize
#### General_SSD / default
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-28", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "SSD",
			"RootDiskSize": "default"
		}
	}' |json_pp


#### RootDiskType / RootDiskSize
#### General_SSD / default
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-29", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_SSD",
			"RootDiskSize": "1100"
		}
	}' |json_pp

