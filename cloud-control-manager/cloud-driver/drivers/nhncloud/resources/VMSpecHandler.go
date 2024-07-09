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

func (vmSpecHandler *NhnCloudVMSpecHandler) mappingVMSpecInfo(vmSpec flavors.Flavor) *irs.VMSpecInfo {
	cblogger.Info("NHN Cloud Cloud Driver: called mappingVMSpecInfo()!")

	vmSpecInfo := &irs.VMSpecInfo{
		Region: vmSpecHandler.RegionInfo.Region,
		Name:   vmSpec.Name,
		VCpu:   irs.VCpuInfo{Count: strconv.Itoa(vmSpec.VCPUs)},
		Mem:    strconv.Itoa(vmSpec.RAM),
		// Gpu:          []irs.GpuInfo{{Count: "N/A", Mfr: "N/A", Model: "N/A", Mem: "N/A"}},

		KeyValueList: []irs.KeyValue{
			{Key: "Region", 		   Value: vmSpecHandler.RegionInfo.Region},
			{Key: "VMSpecType", 	   Value: vmSpec.ExtraSpecs.FlavorType},			
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
