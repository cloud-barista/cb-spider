// Tencent Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Tencent Driver.
//
// by CB-Spider Team, 2022.09.

package TencentTestConfig

import (
	"io/ioutil"

	tdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("Tencent Resource Test")
	cblog.SetLevel("debug")
}

// 환경 설정 파일
type Config struct {
	Tencent struct {
		SecretId  string `yaml:"tencent_secret_id"`
		SecretKey string `yaml:"tencent_secret_key"`
		Region    string `yaml:"region"`
		Zone      string `yaml:"zone"`
	} `yaml:"tencent"`
}

// 환경 설정 파일 읽기
// 환경변수 CBSPIDER_PATH 설정 후 해당 폴더 하위에 /config/configTencent.yaml 파일 생성해야 함.
func ReadConfigFile() Config {
	// Set Environment Value of Project Root Path
	// /mnt/d/Workspace/mcloud-barista-config/config/config.yaml
	//testFilePath := os.Getenv("CBSPIDER_PATH") + "/config/configTencent.yaml" //혹시 모를 키 노출 대비 시스템 외부에 존재(개발용용)
	testFilePath := "./conf/testConfigTencent.yaml"
	cblogger.Debugf("Test Data 설정파일 : [%]", testFilePath)

	data, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	cblogger.Info("Loaded ConfigFile...")
	//spew.Dump(config)
	//cblogger.Info(config)
	cblogger.Debug(config.Tencent.SecretId, " ", config.Tencent.Region)
	//cblogger.Debug(config.Tencent.Region)
	return config
}

// handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
// (예) ImageHandler.go -> "Image"
func GetResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(tdrv.TencentDriver)

	config := ReadConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Tencent.SecretId,
			ClientSecret: config.Tencent.SecretKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Tencent.Region,
			Zone:   config.Tencent.Zone,
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
	//case "Publicip":
	//resourceHandler, err = cloudConnection.CreatePublicIPHandler()
	case "Security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
		//	case "VNetwork":
		//resourceHandler, err = cloudConnection.CreateVNetworkHandler()
	case "KeyPair":
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
	case "VPC":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	//case "VNic":
	//resourceHandler, err = cloudConnection.CreateVNicHandler()
	case "VMSpec":
		resourceHandler, err = cloudConnection.CreateVMSpecHandler()
	case "VM":
		resourceHandler, err = cloudConnection.CreateVMHandler()
	case "NLB":
		resourceHandler, err = cloudConnection.CreateNLBHandler()
	case "Disk":
		resourceHandler, err = cloudConnection.CreateDiskHandler()
	case "MyImage":
		resourceHandler, err = cloudConnection.CreateMyImageHandler()

	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}
