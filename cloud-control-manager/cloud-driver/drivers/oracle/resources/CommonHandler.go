package resources

import (
	"fmt"
	"strings"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/sirupsen/logrus"
)

const (
	oracleProviderName     = "ORACLE"
	defaultVMUserID        = "cb-user"
	oracleCloudInitPath    = "/cloud-driver-libs/.cloud-init-oracle/cloud-init"
	cloudInitPublicKeyVar  = "*PUBLIC_KEY*"
	oracleVMKeyPairNameTag = "CB-Spider-KeyPairName"
	oracleVMUserIDTag      = "CB-Spider-VMUserID"
)

var once sync.Once
var cblogger *logrus.Logger
var calllogger *logrus.Logger

func InitLog() {
	once.Do(func() {
		cblogger = cblog.GetLogger("CB-SPIDER ORACLE")
		calllogger = call.GetLogger("HISCALL")
	})
}

func getCallLogScheme(region idrv.RegionInfo, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	return call.CLOUDLOGSCHEMA{CloudOS: call.ORACLE, RegionZone: region.Region, ResourceType: resourceType, ResourceName: resourceName, CloudOSAPI: apiName}
}

func logError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	if calllogger != nil {
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
	}
}

func logInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	if calllogger != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		calllogger.Info(call.String(hiscallInfo))
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func timeValue(value *common.SDKTime) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.Time
}

func freeformTags(tagList []irs.KeyValue) map[string]string {
	if len(tagList) == 0 {
		return nil
	}
	tags := make(map[string]string, len(tagList))
	for _, tag := range tagList {
		if tag.Key != "" {
			tags[tag.Key] = tag.Value
		}
	}
	return tags
}

func freeformTagsWith(tagList []irs.KeyValue, extraTags map[string]string) map[string]string {
	tags := freeformTags(tagList)
	if tags == nil {
		tags = make(map[string]string, len(extraTags))
	}
	for key, value := range extraTags {
		if key != "" {
			tags[key] = value
		}
	}
	return tags
}

func tagValue(tags map[string]string, key string) string {
	if tags == nil {
		return ""
	}
	return tags[key]
}

func tagList(tags map[string]string) []irs.KeyValue {
	list := make([]irs.KeyValue, 0, len(tags))
	for key, value := range tags {
		list = append(list, irs.KeyValue{Key: key, Value: value})
	}
	return list
}

func dnsLabel(name string) string {
	name = strings.ToLower(name)
	var builder strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	label := builder.String()
	if label == "" || label[0] < 'a' || label[0] > 'z' {
		label = "cb" + label
	}
	if len(label) > 15 {
		label = label[:15]
	}
	return label
}

func idFilter(iid irs.IID) (id *string, displayName *string) {
	if iid.SystemId != "" {
		return common.String(iid.SystemId), nil
	}
	return nil, common.String(iid.NameId)
}

func statusErr(prefix string, err error) error {
	return fmt.Errorf("%s: %w", prefix, err)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "notfound") || strings.Contains(strings.ToLower(err.Error()), "404")
}
