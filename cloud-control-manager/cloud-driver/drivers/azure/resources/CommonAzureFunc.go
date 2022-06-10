package resources

import (
	"errors"
	"fmt"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
)

const (
	CBResourceGroupName  = "CB-GROUP"
	CBVirutalNetworkName = "CB-VNet"
	CBVnetDefaultCidr    = "130.0.0.0/16"
	CBVMUser             = "cb-user"
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
	cblogger.Info(fmt.Sprintf("Call %s %s", call.AZURE, apiName))
	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.AZURE,
		RegionZone:   region.Region,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

// 서브넷 CIDR 생성 (CIDR C class 기준 생성)
/*func CreateSubnetCIDR(subnetList []*irs.VPCHandler) (*string, error) {

	addressPrefix := "0.0.0.0/24"

	// CIDR C class 최대값 찾기
	maxClassNum := 0
	for _, subnet := range subnetList {
		//addressArr := strings.Split(subnet.AddressPrefix, ".")
		addressArr := strings.Split(addressPrefix, ".")
		if curClassNum, err := strconv.Atoi(addressArr[2]); err != nil {
			return nil, err
		} else {
			if curClassNum > maxClassNum {
				maxClassNum = curClassNum
			}
		}
	}

	if len(subnetList) == 0 {
		maxClassNum = 0
	} else {
		maxClassNum = maxClassNum + 1
	}

	// 서브넷 CIDR 할당
	vNetIP := strings.Split(CBVnetDefaultCidr, "/")
	vNetIPClass := strings.Split(vNetIP[0], ".")
	subnetCIDR := fmt.Sprintf("%s.%s.%d.0/24", vNetIPClass[0], vNetIPClass[1], maxClassNum)
	return &subnetCIDR, nil
}*/

func GetVNicIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, vNicName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, vNicName)
}

func GetPublicIPIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, publicIPName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, publicIPName)
}

func GetSecGroupIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, secGroupName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, secGroupName)
}

func GetSshKeyIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, keyName string) string{
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/sshPublicKeys/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, keyName)
}

func GetSshKeyNameById(sshId string) (string, error){
	slice := strings.Split(sshId, "/")
	sliceLen := len(slice)
	for index,item := range slice{
		if item == "sshPublicKeys" && sliceLen > index + 1 {
			return slice[index + 1], nil
		}
	}
	return "", errors.New(fmt.Sprintf("Invalid ResourceName"))
}

func CheckIIDValidation(IId irs.IID) bool {
	if IId.NameId == "" && IId.SystemId == "" {
		return false
	}
	return true
}
