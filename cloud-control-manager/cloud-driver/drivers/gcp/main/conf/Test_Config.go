// Proof of Concepts of CB-Spider.
// The CB-Spider is sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by devunet@mz.co.kr, 2019.08.

package TestConfig

import (
	"encoding/json"
	"os"

	gcpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("GCP Resource Test")
	cblog.SetLevel("debug")
}

// GCP에서 다운로드한 JSON파일 포멧
type Config struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientID     string `json:"client_id"`
	AuthURI      string `json:"auth_uri"`
	TokenURI     string `json:"token_uri"`
	AuthProvider string `json:"auth_provider_x509_cert_url"`
}

type HandlerType string

const (
	Image         = "Image"
	MyImage       = "MyImage"
	SecurityGroup = "SecurityGroup"
	Disk          = "Disk"
	VM            = "VM"
	KeyPair       = "KeyPair"
	VMSpec        = "VMSpec"
	VPC           = "VPC"
	NLB           = "NLB"
	RegionZone    = "RegionZone"
	Price         = "Price"
	Tag           = "Tag"
	Cluster       = "Cluster"
)

const (
	region = "asia-northeast3"
	zone   = "asia-northeast3-c"
	// region = "us-central1"
	// zone   = "us-central1-b"
)

// 환경변수 : GOOGLE_APPLICATION_CREDENTIALS - 인증용 .json 파일의 위치
func GetResourceHandler(handlerType HandlerType) (interface{}, error) {
	cloudDriver := new(gcpdrv.GCPDriver)

	credentialFilePath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	cblogger.Infof("export $GOOGLE_APPLICATION_CREDENTIALS : [%s]", credentialFilePath)
	cblogger.Infof("credentialFilePath : [%s]", credentialFilePath)

	config, _ := readFileConfig(credentialFilePath)

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			PrivateKey:  config.PrivateKey,
			ProjectID:   config.ProjectID,
			ClientEmail: config.ClientEmail,
		},
		RegionInfo: idrv.RegionInfo{
			Region: region,
			Zone:   zone,
		},
	}

	cloudConnection, errCon := cloudDriver.ConnectCloud(connectionInfo)
	if errCon != nil {
		cblogger.Error(errCon)
		return nil, errCon
	}
	cblogger.Info("ConnectCloud Success!!!")

	var resourceHandler interface{}
	var err error

	switch handlerType {
	case Image:
		resourceHandler, err = cloudConnection.CreateImageHandler()
	case MyImage:
		resourceHandler, err = cloudConnection.CreateMyImageHandler()
	case SecurityGroup:
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case Disk:
		resourceHandler, err = cloudConnection.CreateDiskHandler()
	case VM:
		resourceHandler, err = cloudConnection.CreateVMHandler()
	case KeyPair:
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
	case VMSpec:
		resourceHandler, err = cloudConnection.CreateVMSpecHandler()
	case VPC:
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	case NLB:
		resourceHandler, err = cloudConnection.CreateNLBHandler()
	case RegionZone:
		resourceHandler, err = cloudConnection.CreateRegionZoneHandler()
	case Price:
		resourceHandler, err = cloudConnection.CreatePriceInfoHandler()
	case Tag:
		resourceHandler, err = cloudConnection.CreateTagHandler()
	case Cluster:
		resourceHandler, err = cloudConnection.CreateClusterHandler()
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

func readFileConfig(filepath string) (Config, error) {

	data, err := os.ReadFile(filepath)
	if err != nil {
		cblogger.Error(err)
		panic(err)
	}

	var config Config
	json.Unmarshal(data, &config)
	cblogger.Info("Loaded ConfigFile...")

	return config, err

}
