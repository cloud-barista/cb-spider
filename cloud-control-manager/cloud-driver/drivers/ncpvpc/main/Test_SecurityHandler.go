// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// Updated by ETRI, 2024.11.

package main

import (
	"os"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cblog "github.com/cloud-barista/cb-log"

	// ncpvpcdrv "github.com/cloud-barista/ncpvpc/ncpvpc"  // For local test
	ncpvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncpvpc"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC Resource Test")
	cblog.SetLevel("info")
}

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := getResourceHandler("Security")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	//config := readConfigFile()
	//VmID := config.Ncp.VmID

	for {
		cblogger.Info("\n============================================================================================")
		cblogger.Info("[ Security Management Test ]")
		cblogger.Info("1. List Security")
		cblogger.Info("2. Get Security")
		cblogger.Info("3. Create Security")
		cblogger.Info("4. Add Rules")
		cblogger.Info("5. Remove Rules")
		cblogger.Info("6. Delete Security")
		cblogger.Info("7. List IID")
		cblogger.Info("0. Quit")
		cblogger.Info("\n   Select a number above!! : ")
		cblogger.Info("============================================================================================")

		var commandNum int

		securityName := "ncp-sg-006"
		securityId := "78628"
		vpcId := "19368"

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				cblogger.Infof("Exit")
				return

			case 1:
				result, err := handler.ListSecurity()
				if err != nil {
					cblogger.Infof("Failed to retrieve SecurityGroup list: ", err)
				} else {
					cblogger.Info("SecurityGroup list retrieval result")
					//cblogger.Info(result)
					spew.Dump(result)
					cblogger.Infof("=========== S/G count : [%d] ================", len(result))
					if result != nil {
						securityId = result[0].IId.SystemId // Change to the created ID for retrieval and deletion
					}
				}
				cblogger.Info("\nListSecurity() Test Finished")

			case 2:
				cblogger.Infof("[%s] SecurityGroup information retrieval test", securityId)
				result, err := handler.GetSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof("Failed to retrieve SecurityGroup [%s]: %v", securityId, err)
				} else {
					cblogger.Infof("[%s] SecurityGroup retrieval result: [%v]", securityId, result)
					spew.Dump(result)
				}
				cblogger.Info("\nGetSecurity() Test Finished")

			case 3:
				cblogger.Infof("[%s] SecurityGroup creation test", securityName)

				securityReqInfo := irs.SecurityReqInfo{
					IId:    irs.IID{NameId: securityName},
					VpcIID: irs.IID{SystemId: vpcId},
					SecurityRules: &[]irs.SecurityRuleInfo{ // Setting Security Rules
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "20",
						// 	ToPort:     "22",
						// 	CIDR: 		"0.0.0.0/0",
						// },
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "80",
						// 	ToPort:     "80",
						// 	CIDR: 		"172.16.0.0/16",							
						// },
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "-1",
						// 	ToPort:     "-1",
						// 	CIDR: 		"172.16.0.0/16",							
						// },
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "udp",
						// 	FromPort:   "8080",
						// 	ToPort:     "8080",
						// 	CIDR: 		"0.0.0.0/0",
						// },
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "icmp",
						// 	FromPort:   "-1",
						// 	ToPort:     "-1",
						// 	CIDR: 		"0.0.0.0/0",
						// },
						// {
						// 	Direction:  "outbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "443",
						// 	ToPort:     "443",
						// 	CIDR: 		"0.0.0.0/0",
						// },
						// {
						// 	Direction:  "outbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "8443",
						// 	ToPort:     "9999",
						// 	CIDR: 		"172.16.0.0/16",	
						// },

						// Allow all traffic
						{
							Direction:  "inbound",
							IPProtocol: "ALL",
							FromPort:   "-1",
							ToPort:     "-1",
							CIDR: 		"0.0.0.0/0",
						},
						{
							Direction:  "outbound",
							IPProtocol: "ALL",
							FromPort:   "-1",
							ToPort:     "-1",
							CIDR: 		"0.0.0.0/0",
						},
					},
				}

				result, err := handler.CreateSecurity(securityReqInfo)
				if err != nil {
					cblogger.Infof("%s SecurityGroup creation failed: %v", securityName, err)
				} else {
					cblogger.Infof("[%s] SecurityGroup creation result: [%v]", securityName, result)
					spew.Dump(result)
				}
				cblogger.Info("\nCreateSecurity() Test Finished")

			case 4:
				cblogger.Infof("[%s] Security Rule Add Test", securityName)

				securityRuleReqInfo := &[]irs.SecurityRuleInfo{
					{
						Direction:  "inbound",
						IPProtocol: "tcp",
						FromPort:   "20",
						ToPort:     "22",
						CIDR: 		"0.0.0.0/0",
					},
					{
						Direction:  "inbound",
						IPProtocol: "tcp",
						FromPort:   "80",
						ToPort:     "80",
						CIDR: 		"172.16.0.0/16",							
					},
					{
						Direction:  "inbound",
						IPProtocol: "tcp",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"172.16.0.0/16",							
					},
					{
						Direction:  "inbound",
						IPProtocol: "udp",
						FromPort:   "8080",
						ToPort:     "8080",
						CIDR: 		"0.0.0.0/0",
					},
					{
						Direction:  "inbound",
						IPProtocol: "icmp",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
					{
						Direction:  "outbound",
						IPProtocol: "tcp",
						FromPort:   "443",
						ToPort:     "443",
						CIDR: 		"0.0.0.0/0",
					},
					{
						Direction:  "outbound",
						IPProtocol: "tcp",
						FromPort:   "8443",
						ToPort:     "9999",
						CIDR: 		"172.16.0.0/16",	
					},

					// Allow all traffic
					{
						Direction:  "inbound",
						IPProtocol: "ALL",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
					{
						Direction:  "outbound",
						IPProtocol: "ALL",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
				}

				result, err := handler.AddRules(irs.IID{SystemId: securityId}, securityRuleReqInfo)
				if err != nil {
					cblogger.Infof(securityName, " Security Rule Add failed : ", err)
				} else {
					cblogger.Infof("[%s] Security Rule Add result: [%v]", securityName, result)
					spew.Dump(result)
				}
				cblogger.Info("\nAddRules() Test Finished")

			case 5:
				cblogger.Infof("[%s] Security Rule Remove Test", securityName)

				securityRuleReqInfo := &[]irs.SecurityRuleInfo{
					// {
						
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "20",
					// 	ToPort:     "22",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "80",
					// 	ToPort:     "80",
					// 	CIDR: 		"172.16.0.0/16",							
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "-1",
					// 	ToPort:     "-1",
					// 	CIDR: 		"172.16.0.0/16",							
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "udp",
					// 	FromPort:   "8080",
					// 	ToPort:     "8080",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "icmp",
					// 	FromPort:   "-1",
					// 	ToPort:     "-1",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "outbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "443",
					// 	ToPort:     "443",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "outbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "8443",
					// 	ToPort:     "9999",
					// 	CIDR: 		"172.16.0.0/16",	
					// },


					// // All traffic 허용 rule
					{
						Direction:  "inbound",
						IPProtocol: "ALL",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
					{
						Direction:  "outbound",
						IPProtocol: "ALL",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
				}

				result, err := handler.RemoveRules(irs.IID{SystemId: securityId}, securityRuleReqInfo)
				if err != nil {
					cblogger.Infof(securityName, " Security Rule Remove failed : ", err)
				} else {
					cblogger.Infof("[%s] Security Rule removal result: [%v]", securityName, result)
					spew.Dump(result)
				}	
				cblogger.Info("\nRemoveRules() Test Finished")

			case 6:
				cblogger.Infof("[%s] SecurityGroup deletion test", securityId)
				result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof("Failed to delete SecurityGroup [%s]: %v", securityId, err)
				} else {
					cblogger.Infof("[%s] SecurityGroup deletion result: [%s]", securityId, result)
				}
				cblogger.Info("\nDeleteSecurity() Test Finished")

			case 7:
				cblogger.Info("Start ListIID() ...")
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Error("Failed to retrieve S/G IID list: ", err)
				} else {
					cblogger.Info("Successfully retrieved S/G IID list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total number of IID list: [%d]", len(result))
				}
				cblogger.Info("\nListIID() Test Finished")		
			}
		}
	}
}

func testErr() error {
	return errors.New("")
}

func main() {
	cblogger.Info("NCP VPC Resource Test")

	handleSecurity()
}

// handlerType: The string before "Handler" in the xxxHandler.go file in the resources folder
// (e.g., ImageHandler.go -> "Image")
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ncpvpcdrv.NcpVpcDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Ncp.NcpAccessKeyID,
			ClientSecret: config.Ncp.NcpSecretKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Ncp.Region,
			Zone:   config.Ncp.Zone,
		},
	}

	// NOTE Just for test
	cblogger.Info(config.Ncp.NcpAccessKeyID)
	// cblogger.Info(config.Ncp.NcpSecretKey)

	cloudConnection, errCon := cloudDriver.ConnectCloud(connectionInfo)
	if errCon != nil {
		return nil, errCon
	}

	var resourceHandler interface{}
	var err error

	switch handlerType {
	case "Image":
		resourceHandler, err = cloudConnection.CreateImageHandler()
	case "Security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "VNetwork":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	case "VM":
		resourceHandler, err = cloudConnection.CreateVMHandler()
	case "VMSpec":
		resourceHandler, err = cloudConnection.CreateVMSpecHandler()
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

// Region: The region to use (e.g., ap-northeast-2)
// ImageID: The AMI ID to use for VM creation (e.g., ami-047f7b46bd6dd5d84)
// BaseName: The prefix name to use when creating multiple VMs (VMs will be created in the format "BaseName" + "_" + "number") (e.g., mcloud-barista)
// VmID: The EC2 instance ID to test the lifecycle
// InstanceType: The instance type to use when creating a VM (e.g., t2.micro)
// KeyName: The key pair name to use when creating a VM (e.g., mcloud-barista-keypair)
// MinCount: The minimum number of instances to create
// MaxCount: The maximum number of instances to create
// SubnetId: The SubnetId of the VPC where the VM will be created (e.g., subnet-cf9ccf83)
// SecurityGroupID: The security group ID to apply to the created VM (e.g., sg-0df1c209ea1915e4b)
type Config struct {
	Ncp struct {
		NcpAccessKeyID string `yaml:"ncp_access_key_id"`
		NcpSecretKey   string `yaml:"ncp_secret_key"`
		Region         string `yaml:"region"`
		Zone           string `yaml:"zone"`

		ImageID string `yaml:"image_id"`

		VmID         string `yaml:"ncp_instance_id"`
		BaseName     string `yaml:"base_name"`
		InstanceType string `yaml:"instance_type"`
		KeyName      string `yaml:"key_name"`
		MinCount     int64  `yaml:"min_count"`
		MaxCount     int64  `yaml:"max_count"`

		SubnetID        string `yaml:"subnet_id"`
		SecurityGroupID string `yaml:"security_group_id"`

		PublicIP string `yaml:"public_ip"`
	} `yaml:"ncpvpc"`
}

func readConfigFile() Config {
	// # Set Environment Value of Project Root Path
	// goPath := os.Getenv("GOPATH")
	// rootPath := goPath + "/src/github.com/cloud-barista/ncp/ncp/main"
	// cblogger.Debugf("Test Config file : [%]", rootPath+"/config/config.yaml")
	rootPath 	:= os.Getenv("CBSPIDER_ROOT")
	configPath 	:= rootPath + "/cloud-control-manager/cloud-driver/drivers/ncpvpc/main/config/config.yaml"
	cblogger.Debugf("Test Config file : [%s]", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}
	cblogger.Info("ConfigFile Loaded ...")

	// Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
