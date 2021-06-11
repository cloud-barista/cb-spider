package resources

import (
	"crypto/md5"
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

const (
	CBDefaultVmUserName string = "cb-user"
	CBKeyPairPath = "/cloud-driver-libs/.ssh-ibm/"
	productName = "PUBLIC_CLOUD_SERVER"
	CBCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-ibm/cloud-init"
)

const IbmVmStatusRunning = "Running"
const IbmVmStatusHalted = "Halted"
const IbmVmStatusPaused = "Paused"
const IbmVmStatusSuspended = "Suspended"
const IbmVmStatusUnknown = "Unknown"
const staticSubnetPackageKeyName = "STATIC_IP_ADDRESSES"
const potableSubnetPackageKeyName = "PORTABLE_IP_ADDRESSES"
const globalSubnetPackageKeyName = "GLOBAL_IP_ADDRESSES"

var once sync.Once
var cblogger *logrus.Logger
var calllogger *logrus.Logger

func InitLog() {
	once.Do(func() {
		// cblog is a global variable.
		cblogger = cblog.GetLogger("CB-SPIDER")
		calllogger = call.GetLogger("HISCALL")
	})
}

func LoggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	cblogger.Error(err.Error())
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Info(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(region idrv.RegionInfo, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.IBM, apiName))
	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.IBM,
		RegionZone:   region.Region,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

func CreateHashString(credentialInfo idrv.CredentialInfo) (string, error) {
	keyString := credentialInfo.Username + credentialInfo.ApiKey
	hasher := md5.New()
	_, err := io.WriteString(hasher, keyString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func GetPublicKey(credentialInfo idrv.CredentialInfo, keyPairName string) (string, error) {
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString, err := CreateHashString(credentialInfo)
	if err != nil {
		return "", err
	}

	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"
	publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}
	return string(publicKeyBytes), nil
}
