// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, Innogrid, 2021.12.
// by ETRI 2022.03 updated.

package resources

import (
	"fmt"
	"strconv"
	"strings"
	// "errors"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/flavors"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudVMSpecHandler struct {
	RegionInfo idrv.RegionInfo
	VMClient   *nhnsdk.ServiceClient
}

func (vmSpecHandler *NhnCloudVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called ListVMSpec()!")
	callLogInfo := getCallLogScheme(vmSpecHandler.RegionInfo.Region, call.VMSPEC, "ListVMSpec()", "ListVMSpec()")

	listOpts := flavors.ListOpts{
		Limit: 100, // Note) default : 20
	}
	start := call.Start()
	allPages, err := flavors.ListDetail(vmSpecHandler.VMClient, listOpts).AllPages()
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NHN Flavor List Pages : ", err)
		return nil, rtnErr
	}
	specList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Extract NHN Flavor List : ", err)
		return nil, rtnErr
	}
	LoggingInfo(callLogInfo, start)

	var vmSpecInfoList []*irs.VMSpecInfo
	for _, vmSpec := range specList {
		vmSpecInfo := vmSpecHandler.mappingVMSpecInfo(vmSpec)
		vmSpecInfoList = append(vmSpecInfoList, vmSpecInfo)
	}
	return vmSpecInfoList, nil
}

func (vmSpecHandler *NhnCloudVMSpecHandler) GetVMSpec(specName string) (irs.VMSpecInfo, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called GetVMSpec()!")
	callLogInfo := getCallLogScheme(vmSpecHandler.RegionInfo.Region, call.VMSPEC, specName, "GetVMSpec()")

	if strings.EqualFold(specName, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid vmSpec Name!!", "")
		return irs.VMSpecInfo{}, rtnErr
	}

	flavorId, err := vmSpecHandler.getIDFromName(specName)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Flavor ID From the Name : ", err)
		return irs.VMSpecInfo{}, rtnErr
	}
	start := call.Start()
	flavor, err := flavors.Get(vmSpecHandler.VMClient, flavorId).Extract()
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NHN Flavor Info : ", err)
		return irs.VMSpecInfo{}, rtnErr
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(flavor)

	vmSpecInfo := vmSpecHandler.mappingVMSpecInfo(*flavor)
	return *vmSpecInfo, nil
}

func (vmSpecHandler *NhnCloudVMSpecHandler) ListOrgVMSpec() (string, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called ListOrgVMSpec()!")
	callLogInfo := getCallLogScheme(vmSpecHandler.RegionInfo.Region, call.VMSPEC, "ListOrgVMSpec()", "ListOrgVMSpec()")

	start := call.Start()
	allPages, err := flavors.ListDetail(vmSpecHandler.VMClient, flavors.ListOpts{}).AllPages()
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NHN Flavor List Pages : ", err)
		return "", rtnErr
	}
	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Extract Flavor List : ", err)
		return "", rtnErr
	}
	LoggingInfo(callLogInfo, start)

	var flvList struct {
		Result []flavors.Flavor `json:"flavorList"`
	}
	flvList.Result = flavorList

	jsonString, err := convertJsonString(flvList)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert to Json String : ", err)
		return "", rtnErr
	}
	return jsonString, nil
}

func (vmSpecHandler *NhnCloudVMSpecHandler) GetOrgVMSpec(specName string) (string, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called GetOrgVMSpec()!")
	callLogInfo := getCallLogScheme(vmSpecHandler.RegionInfo.Region, call.VMSPEC, specName, "GetOrgVMSpec()")

	if strings.EqualFold(specName, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid vmSpec Name!!", "")
		return "", rtnErr
	}

	flavorId, err := vmSpecHandler.getIDFromName(specName)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Flavor ID From the Name : ", err)
		return "", rtnErr
	}
	start := call.Start()
	flavor, err := flavors.Get(vmSpecHandler.VMClient, flavorId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Flavor Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	LoggingInfo(callLogInfo, start)

	var nhnFlavor struct {
		Result flavors.Flavor `json:"flavor"`
	}
	nhnFlavor.Result = *flavor

	jsonString, err := convertJsonString(nhnFlavor)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert to Json String : ", err)
		return "", rtnErr
	}
	return jsonString, nil
}

func getGpuCount(vmSize string) (ea int, memory int) {
	vmSize = strings.ToLower(vmSize)

	// https://www.nhncloud.com/kr/pricing/m-content?c=Machine%20Learning&s=AI%20EasyMaker
	if strings.Contains(vmSize, "g2") {
		if strings.Contains(vmSize, "v100") {
			if strings.Contains(vmSize, "c8m90") {
				return 1, 32
			} else if strings.Contains(vmSize, "c16m180") {
				return 2, 64
			} else if strings.Contains(vmSize, "c32m360") {
				return 4, 128
			} else if strings.Contains(vmSize, "c64m720") {
				return 8, 256
			}
		} else if strings.Contains(vmSize, "t4") {
			if strings.Contains(vmSize, "c4m32") {
				return 1, 16
			} else if strings.Contains(vmSize, "c8m64") {
				return 2, 32
			} else if strings.Contains(vmSize, "c16m128") {
				return 4, 64
			} else if strings.Contains(vmSize, "c32m256") {
				return 8, 128
			}
		}
	} else if strings.Contains(vmSize, "g4") {
		if strings.Contains(vmSize, "c92m1800") {
			return 8, 320
		}
	}

	return -1, -1
}

func getGpuModel(vmSize string) string {
	vmSize = strings.ToLower(vmSize)

	if strings.Contains(vmSize, ".v100.") {
		return "Tesla V100"
	} else if strings.Contains(vmSize, ".t4.") {
		return "Tesla T4"
	} else if strings.Contains(vmSize, "c92m1800") {
		return "A100"
	}

	return ""
}

func parseGpuInfo(vmSizeName string) *irs.GpuInfo {
	vmSizeLower := strings.ToLower(vmSizeName)

	// Check if it's a GPU series
	if !strings.HasPrefix(vmSizeLower, "g2") &&
		!strings.HasPrefix(vmSizeLower, "g4") {
		return nil
	}

	count, mem := getGpuCount(vmSizeLower)
	model := getGpuModel(vmSizeLower)

	return &irs.GpuInfo{
		Count: fmt.Sprintf("%d", count),
		Mem:   fmt.Sprintf("%d", mem*1024),
		Mfr:   "NVIDIA",
		Model: model,
	}
}

func (vmSpecHandler *NhnCloudVMSpecHandler) mappingVMSpecInfo(vmSpec flavors.Flavor) *irs.VMSpecInfo {
	cblogger.Info("NHN Cloud Cloud Driver: called mappingVMSpecInfo()!")

	gpuInfoList := make([]irs.GpuInfo, 0)
	gpuInfo := parseGpuInfo(vmSpec.Name)
	if gpuInfo != nil {
		gpuInfoList = append(gpuInfoList, *gpuInfo)
	}

	vmSpecInfo := &irs.VMSpecInfo{
		Region: vmSpecHandler.RegionInfo.Region,
		Name:   vmSpec.Name,
		VCpu:   irs.VCpuInfo{Count: strconv.Itoa(vmSpec.VCPUs)},
		Mem:    strconv.Itoa(vmSpec.RAM),
		Disk:   strconv.Itoa(vmSpec.Disk),
		Gpu:    gpuInfoList,

		KeyValueList: []irs.KeyValue{
			{Key: "Region", Value: vmSpecHandler.RegionInfo.Region},
			{Key: "VMSpecType", Value: vmSpec.ExtraSpecs.FlavorType},
			{Key: "LocalDiskSize(GB)", Value: strconv.Itoa(vmSpec.Disk)},
		},
	}

	if strings.EqualFold(strconv.Itoa(vmSpec.Disk), "0") {
		keyValue := irs.KeyValue{
			Key:   "Notice!!",
			Value: "Specify 'RootDiskType' and 'RootDiskSize' when VM Creation to Boot from the Attached Volume!!",
		}
		vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList, keyValue)
	}
	return vmSpecInfo
}

func (vmSpecHandler *NhnCloudVMSpecHandler) getIDFromName(specName string) (string, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called getIDFromName()!")

	allPages, err := flavors.ListDetail(vmSpecHandler.VMClient, flavors.ListOpts{}).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Flavor List Pages : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract NHN Flavor List : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var flavorNameList []flavors.Flavor
	for _, flavor := range flavorList {
		if strings.EqualFold(flavor.Name, specName) {
			flavorNameList = append(flavorNameList, flavor)
		}
	}

	if len(flavorNameList) > 1 {
		return "", fmt.Errorf("Found multiple vmSpec with the name %s", specName)
	} else if len(flavorNameList) == 0 {
		return "", fmt.Errorf("Failed to Find vmSpec with the name %s", specName)
	}
	return flavorNameList[0].ID, nil
}
