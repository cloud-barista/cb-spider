package resources

import (
	"errors"
	"fmt"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
)

const (
	CBResourceGroupName  = "CB-GROUP"
	CBVirtualNetworkName = "CB-VNet"
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

func GetResourceNameById(id string) string {
	idArr := strings.Split(id, "/")
	if len(idArr) < 2 {
		return ""
	}
	return idArr[len(idArr)-1]
}

type AzureResourceCategory string

type AzureResourceKind string

const (
	AzureNetworkCategory AzureResourceCategory = "Microsoft.Network"
	AzureComputeCategory AzureResourceCategory = "Microsoft.Compute"

	AzureVirtualNetworks          AzureResourceKind = "virtualNetworks"
	AzureSSHPublicKeys            AzureResourceKind = "sshPublicKeys"
	AzureSecurityGroups           AzureResourceKind = "networkSecurityGroups"
	AzurePublicIPAddresses        AzureResourceKind = "publicIPAddresses"
	AzureFrontendIPConfigurations AzureResourceKind = "frontendIPConfigurations"
	AzureLoadBalancers            AzureResourceKind = "loadBalancers"
	AzureNetworkInterfaces        AzureResourceKind = "networkInterfaces"
)

func generateRandName(prefix string) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s-%s", prefix, strconv.FormatInt(rand.Int63n(1000000), 10))
}

func GetNetworksResourceIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, resourceKind AzureResourceKind, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, AzureNetworkCategory, resourceKind, name)
}

func GetSecGroupIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, secGroupName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, AzureNetworkCategory, AzureSecurityGroups, secGroupName)
}

func GetSshKeyIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, keyName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, AzureNetworkCategory, AzureSSHPublicKeys, keyName)
}

func GetSshKeyNameById(sshId string) (string, error) {
	slice := strings.Split(sshId, "/")
	sliceLen := len(slice)
	for index, item := range slice {
		if item == "sshPublicKeys" && sliceLen > index+1 {
			return slice[index+1], nil
		}
	}
	return "", errors.New(fmt.Sprintf("Invalid ResourceName"))
}

func GetVMNameById(vmId string) (string, error) {
	slice := strings.Split(vmId, "/")
	sliceLen := len(slice)
	for index, item := range slice {
		if item == "virtualMachines" && sliceLen > index+1 {
			return slice[index+1], nil
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

// VMBootDiskType
func GetVMDiskTypeInitType(diskType string) compute.StorageAccountTypes {
	switch diskType {
	case PremiumSSD:
		return compute.StorageAccountTypesPremiumLRS
	case StandardSSD:
		return compute.StorageAccountTypesStandardSSDLRS
	case StandardHHD:
		return compute.StorageAccountTypesStandardLRS
	default:
		return compute.StorageAccountTypesPremiumLRS
	}
}

// VMBootDiskType
func GetVMDiskInfoType(diskType compute.StorageAccountTypes) string {
	switch diskType {
	case compute.StorageAccountTypesPremiumLRS:
		return PremiumSSD
	case compute.StorageAccountTypesStandardSSDLRS:
		return StandardSSD
	case compute.StorageAccountTypesStandardLRS:
		return StandardHHD
	default:
		return string(diskType)
	}
}

// DiskType
func GetDiskTypeInitType(diskType string) compute.DiskStorageAccountTypes {
	switch diskType {
	case PremiumSSD:
		return compute.DiskStorageAccountTypesPremiumLRS
	case StandardSSD:
		return compute.DiskStorageAccountTypesStandardSSDLRS
	case StandardHHD:
		return compute.DiskStorageAccountTypesStandardLRS
	default:
		return compute.DiskStorageAccountTypesPremiumLRS
	}
}

// DiskType
func GetDiskInfoType(diskType compute.DiskStorageAccountTypes) string {
	switch diskType {
	case compute.DiskStorageAccountTypesPremiumLRS:
		return PremiumSSD
	case compute.DiskStorageAccountTypesStandardSSDLRS:
		return StandardSSD
	case compute.DiskStorageAccountTypesStandardLRS:
		return StandardHHD
	default:
		return string(diskType)
	}
}
