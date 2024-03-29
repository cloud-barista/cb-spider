// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2022.09.

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

func handleDisk() {
	cblogger.Debug("Start Disk Resource Test")

	resourceHandler, err := getResourceHandler("Disk")
	if err != nil {
		cblogger.Error(err)
		return
	}
	diskHandler := resourceHandler.(irs.DiskHandler)
	//config := readConfigFile()

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ DiskHandler Test ]")
		cblogger.Info("1. ListDisk()")
		cblogger.Info("2. GetDisk()")
		cblogger.Info("3. CreateDisk()")
		cblogger.Info("4. DeleteDisk()")
		cblogger.Info("5. ChangeDiskSize()")
		cblogger.Info("6. AttachDisk()")
		cblogger.Info("7. DetachDisk()")
		cblogger.Info("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int

		diskIId := irs.IID{
			SystemId: "14812735",
		}

		createReqInfo := irs.DiskInfo{
			IId: irs.IID{
				NameId: "ncpvpc-disk-05",
			},
			// DiskType: "default",
			DiskType: "HDD",
			// DiskType: "SSD",
			// DiskSize: "default",
			DiskSize: "50",
		}

		vmIId := irs.IID{  // To attach disk
			SystemId: "13149859",
		}

		newDiskSize := "160"

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
			case 1:
				cblogger.Info("Start ListDisk() ...")
				if listResult, err := diskHandler.ListDisk(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listResult)
					cblogger.Info("# 출력 결과 수 : ", len(listResult))
				}
				cblogger.Info("Finish ListDisk()")
			case 2:
				cblogger.Info("Start GetDisk() ...")
				if diskInfo, err := diskHandler.GetDisk(diskIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish GetDisk()")
			case 3:
				cblogger.Info("Start CreateDisk() ...")
				if diskInfo, err := diskHandler.CreateDisk(createReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish CreateDisk()")
			case 4:
				cblogger.Info("Start DeleteDisk() ...")
				if delResult, err := diskHandler.DeleteDisk(diskIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(delResult)
				}
				cblogger.Info("Finish DeleteDisk()")
			case 5:
				cblogger.Info("Start ChangeDiskSize() ...")
				if diskInfo, err := diskHandler.ChangeDiskSize(diskIId, newDiskSize); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish ChangeDiskSize()")
			case 6:
				cblogger.Info("Start AttachDisk() ...")
				if diskInfo, err := diskHandler.AttachDisk(diskIId, vmIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish AttachDisk()")
			case 7:
				cblogger.Info("Start DetachDisk() ...")
				if result, err := diskHandler.DetachDisk(diskIId, vmIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				cblogger.Info("Finish DetachDisk()")		
			}
		}
	}
}

func testErr() error {

	return errors.New("")
}

func main() {
	cblogger.Info("NCP VPC Resource Test")
	handleDisk()
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
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
	case "VPC":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	case "Disk":
		resourceHandler, err = cloudConnection.CreateDiskHandler()
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

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
