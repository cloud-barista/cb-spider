package resources

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
)

const (
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
	AzureNetworkCategory          AzureResourceCategory = "Microsoft.Network"
	AzureComputeCategory          AzureResourceCategory = "Microsoft.Compute"
	AzureContainerServiceCategory AzureResourceCategory = "Microsoft.ContainerService"

	AzureVirtualNetworks          AzureResourceKind = "virtualNetworks"
	AzureSubnet                   AzureResourceKind = "subnets"
	AzureSSHPublicKeys            AzureResourceKind = "sshPublicKeys"
	AzureSecurityGroups           AzureResourceKind = "networkSecurityGroups"
	AzurePublicIPAddresses        AzureResourceKind = "publicIPAddresses"
	AzureFrontendIPConfigurations AzureResourceKind = "frontendIPConfigurations"
	AzureLoadBalancers            AzureResourceKind = "loadBalancers"
	AzureNetworkInterfaces        AzureResourceKind = "networkInterfaces"
	AzureContainerService         AzureResourceKind = "managedClusters"
)

func generateRandName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, strconv.FormatInt(rand.Int63n(1000000), 10))
}

func GetNetworksResourceIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, resourceKind AzureResourceKind, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s", credentialInfo.SubscriptionId, regionInfo.Region, AzureNetworkCategory, resourceKind, name)
}

func GetSecGroupIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, secGroupName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s", credentialInfo.SubscriptionId, regionInfo.Region, AzureNetworkCategory, AzureSecurityGroups, secGroupName)
}

func GetSshKeyIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, keyName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s", credentialInfo.SubscriptionId, regionInfo.Region, AzureComputeCategory, AzureSSHPublicKeys, keyName)
}

func GetClusterIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, clusterName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s", credentialInfo.SubscriptionId, regionInfo.Region, AzureContainerServiceCategory, AzureContainerService, clusterName)
}

func getNodePoolIdByName(credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, clusterName string, nodePoolName string) string {
	return fmt.Sprintf("%s/agentPools/%s", GetClusterIdByName(credentialInfo, regionInfo, clusterName), nodePoolName)
}

func getSubscriptionsById(resourceId string) (string, error) {
	slice := strings.Split(resourceId, "/")
	sliceLen := len(slice)
	for index, item := range slice {
		if item == "subscriptions" && sliceLen > index+1 {
			return slice[index+1], nil
		}
	}
	return "", errors.New(fmt.Sprintf("Invalid ResourceID"))
}

func GetVPCNameById(vpcId string) (string, error) {
	slice := strings.Split(vpcId, "/")
	sliceLen := len(slice)
	for index, item := range slice {
		if item == string(AzureVirtualNetworks) && sliceLen > index+1 {
			return slice[index+1], nil
		}
	}
	return "", errors.New(fmt.Sprintf("Invalid ResourceID"))
}

func GetClusterNameById(clusterId string) (string, error) {
	slice := strings.Split(clusterId, "/")
	sliceLen := len(slice)
	for index, item := range slice {
		if item == string(AzureContainerService) && sliceLen > index+1 {
			return slice[index+1], nil
		}
	}
	return "", errors.New(fmt.Sprintf("Invalid ResourceID"))
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

func getNameById(sshId string, kind AzureResourceKind) (string, error) {
	slice := strings.Split(sshId, "/")
	sliceLen := len(slice)
	for index, item := range slice {
		if item == string(kind) && sliceLen > index+1 {
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

func getResourceGroupById(vmId string) (string, error) {
	slice := strings.Split(vmId, "/")
	sliceLen := len(slice)
	for index, item := range slice {
		if strings.ToLower(item) == "resourcegroups" && sliceLen > index+1 {
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
func GetVMDiskTypeInitType(diskType string) armcompute.StorageAccountTypes {
	switch diskType {
	case PremiumSSD:
		return armcompute.StorageAccountTypesPremiumLRS
	case StandardSSD:
		return armcompute.StorageAccountTypesStandardSSDLRS
	case StandardHDD:
		return armcompute.StorageAccountTypesStandardLRS
	default:
		return armcompute.StorageAccountTypesPremiumLRS
	}
}

// VMBootDiskType
func GetVMDiskInfoType(diskType *armcompute.StorageAccountTypes) string {
	if diskType == nil {
		return "Unknown"
	}

	switch *diskType {
	case armcompute.StorageAccountTypesPremiumLRS:
		return PremiumSSD
	case armcompute.StorageAccountTypesStandardSSDLRS:
		return StandardSSD
	case armcompute.StorageAccountTypesStandardLRS:
		return StandardHDD
	default:
		return string(*diskType)
	}
}

// DiskType
func GetDiskTypeInitType(diskType string) (armcompute.DiskStorageAccountTypes, error) {
	switch diskType {
	case "":
		return armcompute.DiskStorageAccountTypesPremiumLRS, nil
	case "default":
		return armcompute.DiskStorageAccountTypesPremiumLRS, nil
	case PremiumSSD:
		return armcompute.DiskStorageAccountTypesPremiumLRS, nil
	case StandardSSD:
		return armcompute.DiskStorageAccountTypesStandardSSDLRS, nil
	case StandardHDD:
		return armcompute.DiskStorageAccountTypesStandardLRS, nil
	default:
		return "", errors.New(fmt.Sprintf("invalid DiskType %s, Please select one of %s, %s, %s", diskType, PremiumSSD, StandardSSD, StandardHDD))
	}
}

// DiskType
func GetDiskInfoType(diskType armcompute.DiskStorageAccountTypes) string {
	switch diskType {
	case armcompute.DiskStorageAccountTypesPremiumLRS:
		return PremiumSSD
	case armcompute.DiskStorageAccountTypesStandardSSDLRS:
		return StandardSSD
	case armcompute.DiskStorageAccountTypesStandardLRS:
		return StandardHDD
	default:
		return string(diskType)
	}
}

func GetScaleSetVMDiskInfoType(diskType armcompute.StorageAccountTypes) string {
	switch diskType {
	case armcompute.StorageAccountTypesPremiumLRS:
		return PremiumSSD
	case armcompute.StorageAccountTypesStandardSSDLRS:
		return StandardSSD
	case armcompute.StorageAccountTypesStandardLRS:
		return StandardHDD
	default:
		return string(diskType)
	}
}

func overlapCheckCidr(cidr1 string, cidr2 string) (bool, error) {
	cidr1IP, cidr1IPnet, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false, err
	}
	cidr2IP, cidr2IPnet, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false, err
	}
	check1 := cidr1IPnet.Contains(cidr2IP)
	check2 := cidr2IPnet.Contains(cidr1IP)
	return !check1 && !check2, nil
}

func removeDuplicateStr(array []string) []string {
	if len(array) < 1 {
		return array
	}

	sort.Strings(array)
	prev := 1
	for curr := 1; curr < len(array); curr++ {
		if array[curr-1] != array[curr] {
			array[prev] = array[curr]
			prev++
		}
	}

	return array[:prev]
}

func setTags(tagList []irs.KeyValue) map[string]*string {
	tags := make(map[string]*string)
	for _, tag := range tagList {
		tags[tag.Key] = &tag.Value
	}
	return tags
}

func setTagList(tags map[string]*string) []irs.KeyValue {
	tagList := make([]irs.KeyValue, 0, len(tags))
	if len(tags) != 0 {
		for key, value := range tags {
			if value != nil {
				tagList = append(tagList, irs.KeyValue{
					Key:   key,
					Value: *value,
				})
			}
		}
		return tagList
	}
	return nil
}

func GetSubnetIdByName(credentialInfo idrv.CredentialInfo, resourceGroup string, vnetName string, subnetName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		credentialInfo.SubscriptionId,
		resourceGroup,
		vnetName,
		subnetName,
	)
}

func toStrPtr(input string) *string {
	return &input
}

func toInt32Ptr(input int) *int32 {
	inputInt32 := int32(input)

	return &inputInt32
}

func toBoolPtr(input bool) *bool {
	return &input
}

const (
	AzureStorageAccountPrefix = "cbspider"
)
