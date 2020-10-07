package resources

import (
	"fmt"
	"strings"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/nic"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/specs"
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

func LoggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	cblogger.Error(err.Error())
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Info(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(endpoint string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.CLOUDIT, apiName))
	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.CLOUDIT,
		RegionZone:   endpoint,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

// VM Spec 정보 조회
func GetVMSpecByName(authHeader map[string]string, reqClient *client.RestClient, specName string) (*string, error) {
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	specList, err := specs.List(reqClient, &requestOpts)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get security group list, err : %s", err))
		return nil, err
	}

	specInfo := specs.VMSpecInfo{}
	for _, s := range *specList {
		if strings.EqualFold(specName, s.Name) {
			specInfo = s
			break
		}
	}

	// VM Spec 정보가 없을 경우 에러 리턴
	if specInfo.Id == "" {
		cblogger.Error(fmt.Sprintf("failed to get image, err : %s", err))
		return nil, err
	}
	return &specInfo.Id, nil
}

// VNic 목록 조회
func ListVNic(authHeader map[string]string, reqClient *client.RestClient, vmId string) (*[]nic.VmNicInfo, error) {
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	vNicList, err := nic.List(reqClient, vmId, &requestOpts)
	if err != nil {
		return nil, err
	}
	return vNicList, nil
}
