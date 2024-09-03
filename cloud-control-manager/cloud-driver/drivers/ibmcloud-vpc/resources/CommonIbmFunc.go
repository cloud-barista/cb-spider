package resources

import (
	"errors"
	"fmt"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

const (
	CBDefaultVmUserName string = "cb-user"
	CBCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-ibm/cloud-init"
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

func generateRandName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, strconv.FormatInt(rand.Int63n(1000000), 10))
}

func addTag(tagService *globaltaggingv1.GlobalTaggingV1, tag irs.KeyValue, CRN string) error {
	resourceModel := globaltaggingv1.Resource{
		ResourceID: &CRN,
	}

	var tagName string
	if tag.Value == "" {
		tagName = tag.Key
	} else {
		tagName = tag.Key + ":" + tag.Value
	}

	attachTagOptions := tagService.NewAttachTagOptions(
		[]globaltaggingv1.Resource{resourceModel},
	)

	attachTagOptions.SetTagNames([]string{tagName})
	attachTagOptions.SetTagType("user")

	_, _, err := tagService.AttachTag(attachTagOptions)
	if err != nil {
		return err
	}

	return nil
}

func deleteUnusedTags(tagService *globaltaggingv1.GlobalTaggingV1) {
	deleteTagAllOptions := tagService.NewDeleteTagAllOptions()
	deleteTagAllOptions.SetTagType("user")

	_, _, err := tagService.DeleteTagAll(deleteTagAllOptions)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Subnet Detached Tag err = %s", err.Error()))
		cblogger.Error(delErr.Error())
	}
}
