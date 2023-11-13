// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC Connection Driver
//
// by ETRI, 2020.10.

package resources

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	// "strconv"
	"math/rand"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

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

// Convert Cloud Object to JSON String type
func ConvertJsonString(v interface{}) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert Json to String. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return "", newErr
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

func LoggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	cblogger.Error(err.Error())
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Error(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(zoneInfo string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.NCPVPC, apiName))

	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.NCPVPC,
		RegionZone:   zoneInfo,
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

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())
	
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
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
