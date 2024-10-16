// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Driver PoC
//
// by ETRI, 2021.05.

package resources

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"os"
	"strings"
	// "github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	cblog "github.com/cloud-barista/cb-log"	
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

var once sync.Once
var cblogger *logrus.Logger
var calllogger *logrus.Logger

//Cloud Object를 JSON String 타입으로 변환
func ConvertJsonString(v interface{}) (string, error) {
	jsonBytes, jsonErr := json.Marshal(v)
	if jsonErr != nil {
		cblogger.Error("JSON 변환 실패")
		cblogger.Error(jsonErr)
		return "", jsonErr
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

// int32 to string 변환 : String(), int64 to string 변환 : strconv.Itoa()
func String(n int32) string {
	buf := [11]byte{}
	pos := len(buf)
	i := int64(n)
	signed := i < 0
	if signed {
		i = -i
	}
	for {
		pos--
		buf[pos], i = '0'+byte(i%10), i/10
		if i == 0 {
			if signed {
				pos--
				buf[pos] = '-'
			}
			return string(buf[pos:])
		}
	}
}

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
	calllogger.Error(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(zone string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.KTCLOUD, apiName))

	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.KTCLOUD,
		RegionZone:   zone,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

func logAndReturnError(callLogInfo call.CLOUDLOGSCHEMA, givenErrString string, v interface{}) (error) {
	newErr := fmt.Errorf(givenErrString + " %v", v)
	cblogger.Error(newErr.Error())
	LoggingError(callLogInfo, newErr)
	return newErr
}

func CheckFolder(folderPath string) error {
	// Check if the KeyPair Folder Exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
	}

	return nil
}

func CheckFolderAndCreate(folderPath string) error {
	// Check if the Folder Exists and Create it
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if err := os.Mkdir(folderPath, 0700); err != nil {
			return err
		}
	}
	return nil
}

func convertTimeFormat(inputTime string) (time.Time, error) {
	// Parse the input time using the given layout
	layout := "2006-01-02T15:04:05-0700"
	parsedTime, err := time.Parse(layout, inputTime)
	if err != nil {
		newErr := fmt.Errorf("Failed to Parse the Input Time Format!!")
		cblogger.Error(newErr.Error())
		return time.Time{}, newErr
	}
	return parsedTime, nil
}

func createClient(connectionInfo idrv.ConnectionInfo) (*ktsdk.KtCloudClient, error) {
	cblogger.Info("KT Cloud Driver: called createClient()")
	// cblogger.Infof("### connectionInfo.RegionInfo.Zone : [%d]", connectionInfo.RegionInfo.Zone)
	
	// $$$ Caution!!
	var apiurl string
	if strings.EqualFold(connectionInfo.RegionInfo.Zone, KOR_Seoul_M2_ZoneID) { // When Zone is "KOR-Seoul M2"
	apiurl = "https://api.ucloudbiz.olleh.com/server/v2/client/api"
	} else {
	apiurl = "https://api.ucloudbiz.olleh.com/server/v1/client/api"
	}

	if len(apiurl) == 0 {
		newErr := fmt.Errorf("KT Cloud API URL Not Found!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	apikey := connectionInfo.CredentialInfo.ClientId
	if len(apikey) == 0 {
		newErr := fmt.Errorf("KT Cloud API Key Not Found!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	secretkey := connectionInfo.CredentialInfo.ClientSecret
	if len(secretkey) == 0 {
		newErr := fmt.Errorf("KT Cloud Secret Key Not Found!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Always validate any SSL certificates in the chain
	insecureskipverify := false
	client := ktsdk.KtCloudClient{}.New(apiurl, apikey, secretkey, insecureskipverify)

	return client, nil
}

func getSeoulCurrentTime() string {
	loc, _ := time.LoadLocation("Asia/Seoul")
	currentTime := time.Now().In(loc)	
	return currentTime.Format("2006-01-02 15:04:05")
}
