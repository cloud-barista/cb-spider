package resources

import (
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
	err := tagValidation(tag)
	if err != nil {
		return err
	}

	return attachOrDetachTag(tagService, tag, CRN, "add")
}

func deleteUnusedTags(tagService *globaltaggingv1.GlobalTaggingV1) {
	// It only cleans unused tags in IBM cloud user account.
	// Not needed for checking errors and just wait for a long time for the resource deletion to complete.
	go func(tagService *globaltaggingv1.GlobalTaggingV1) {
		time.Sleep(time.Second * 60)
		deleteTagAllOptions := tagService.NewDeleteTagAllOptions()
		deleteTagAllOptions.SetTagType("user")
		_, _, _ = tagService.DeleteTagAll(deleteTagAllOptions)
	}(tagService)
}
