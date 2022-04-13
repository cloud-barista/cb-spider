echo "#########################################################################"
echo "## VM RootDisk Type and Resize Test Scripts for CB-Spider - 2022.04.12."
echo "#########################################################################"

# $$$ In case of SPEC_NAME=u2.c2m4 $$$

#### RootDiskType / RootDiskSize
#### "" / ""
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { 
			"Name": "nhn-vm-10", 
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
			"Name": "nhn-vm-11", 
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
#### General_HDD / 50
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": { 
			"Name": "nhn-vm-12", 
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
#### General_HDD / 20  ==> Must Fail
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-13", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_HDD",
			"RootDiskSize": "20"
		}
	}' |json_pp

#### RootDiskType / RootDiskSize
#### General_SSD / 50  ==> Must Fail
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-14", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_SSD",
			"RootDiskSize": "50"
		}
	}' |json_pp


#### RootDiskType / RootDiskSize
#### TYPE1 / default
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-15", 
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
#### General_SSD / 25  ==> Must Fail
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-16", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "General_SSD",
			"RootDiskSize": "25"
		}
	}' |json_pp

#### RootDiskType / RootDiskSize
#### General_SSD / 25  ==> Must Fail
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'",
		"ReqInfo": {
			"Name": "nhn-vm-17", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VPCName": "nhn-vpc-1", 
			"SubnetName": "nhn-subnet-1", 
			"SecurityGroupNames": [ "nhn-sg-02" ], 
			"VMSpecName": "'${SPEC_NAME}'", 
			"KeyPairName": "nhn-key-01",
			"RootDiskType": "HDD",
			"RootDiskSize": "50"
		}
	}' |json_pp
