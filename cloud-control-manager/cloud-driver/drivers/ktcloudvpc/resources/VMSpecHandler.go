// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI 2022.08.

package resources

import (
	"fmt"
	"strconv"
	"strings"
	"github.com/davecgh/go-spew/spew"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/flavors"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcVMSpecHandler struct {
	RegionInfo idrv.RegionInfo
	VMClient   *ktvpcsdk.ServiceClient
}

func (vmSpecHandler *KTVpcVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListVMSpec()!")	
	callLogInfo := getCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, "ListVMSpec()", "ListVMSpec()")

	listOpts :=	flavors.ListOpts{
		Limit: 300,  //default : 20
	}

	start := call.Start()
	allPages, err := flavors.ListDetail(vmSpecHandler.VMClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Flavor Pages from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Flavor list from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	// cblogger.Infof("\n # specList : ")
	// spew.Dump(specList)

	var vmSpecInfoList []*irs.VMSpecInfo
    for _, flavor := range flavorList {
		vmSpecInfo := vmSpecHandler.mappingVMSpecInfo(&flavor)
		vmSpecInfoList = append(vmSpecInfoList, vmSpecInfo)
    }
	return vmSpecInfoList, nil
}

func (vmSpecHandler *KTVpcVMSpecHandler) GetVMSpec(specName string) (irs.VMSpecInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetVMSpec()!")	
	callLogInfo := getCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, specName, "GetVMSpec()")

	if strings.EqualFold(specName,"") {
		newErr := fmt.Errorf("Invalid vmSpec Name!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMSpecInfo{}, newErr
	}

	vmSpecId, err := vmSpecHandler.getVMSpecIdWithName(specName)
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return irs.VMSpecInfo{}, err
	}

	start := call.Start()
	vmSpec, err := flavors.Get(vmSpecHandler.VMClient, vmSpecId).Extract()
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return irs.VMSpecInfo{}, err
	}
	loggingInfo(callLogInfo, start)
	vmSpecInfo := vmSpecHandler.mappingVMSpecInfo(vmSpec)	
	return *vmSpecInfo, nil
}

func (vmSpecHandler *KTVpcVMSpecHandler) ListOrgVMSpec() (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListOrgVMSpec()!")	
	callLogInfo := getCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, "ListOrgVMSpec()", "ListOrgVMSpec()")

	var vmSpecInfoList []*irs.VMSpecInfo
	vmSpecInfoList, err := vmSpecHandler.ListVMSpec()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VMSpec Info list : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	jsonString, jsonErr := convertJsonString(vmSpecInfoList)
	if jsonErr != nil {
		newErr := fmt.Errorf("Failed to Convert the Json String : [%v]", jsonErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	return jsonString, jsonErr
}

func (vmSpecHandler *KTVpcVMSpecHandler) GetOrgVMSpec(specName string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetOrgVMSpec()!")	
	callLogInfo := getCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, specName, "GetOrgVMSpec()")

	if strings.EqualFold(specName,"") {
		newErr := fmt.Errorf("Invalid vmSpec Name!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	specInfo, err := vmSpecHandler.GetVMSpec(specName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VMSpec Info : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	jsonString, jsonErr := convertJsonString(specInfo)
	if jsonErr != nil {
		newErr := fmt.Errorf("Failed to Convert the Json String : [%v]", jsonErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	return jsonString, jsonErr
}

func (vmSpecHandler *KTVpcVMSpecHandler) getVMSpecIdWithName(specName string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getVMSpecIdWithName()!")

	allPages, err := flavors.ListDetail(vmSpecHandler.VMClient, flavors.ListOpts{}).AllPages()
	if err != nil {
		return "", err
	}
	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return "", err
	}

	var flavorNameList []flavors.Flavor
	for _, flavor := range flavorList {
		if strings.EqualFold(flavor.Name, specName) {
			flavorNameList = append(flavorNameList, flavor)
		}
	}
	if len(flavorNameList) < 1 {
		return "", fmt.Errorf("Found multiple vmSpec with the name %s", specName)
	} else if len(flavorNameList) == 0 {
		return "", fmt.Errorf("Failed to Find vmSpec with the name %s", specName)
	}
	return flavorNameList[0].ID, nil
}

func (vmSpecHandler *KTVpcVMSpecHandler) mappingVMSpecInfo(flavor *flavors.Flavor) *irs.VMSpecInfo {
	cblogger.Info("KT Cloud VPC Driver: called mappingVMSpecInfo()!")
	cblogger.Info("\n\n### flavor : ")
	spew.Dump(flavor)

	vmSpecInfo := irs.VMSpecInfo {
		Region:       vmSpecHandler.RegionInfo.Zone,
		Name:         flavor.Name,
		VCpu:         irs.VCpuInfo{Count: strconv.Itoa(flavor.VCPUs), Clock: "-1"},
		Mem:          strconv.Itoa(flavor.RAM),
		Gpu:          []irs.GpuInfo{{Count: "-1", Mfr: "NA", Model: "NA", Mem: "-1"}},
		Disk: 		  "NA",

		KeyValueList: []irs.KeyValue{
			{Key: "Zone", Value: vmSpecHandler.RegionInfo.Zone},
			// {Key: "RootDiskSize(GB)", Value: strconv.Itoa(flavor.Disk)},
			// {Key: "EphemeralDiskSize(GB)", Value: strconv.Itoa(flavor.Ephemeral)},
			// {Key: "SwapDiskSize(MB)", Value: strconv.Itoa(flavor.Swap)},
			{Key: "IsPublic", Value: strconv.FormatBool(flavor.IsPublic)},
			{Key: "VMSpecID", Value: flavor.ID},
		},
	}

	// if strings.EqualFold(strconv.Itoa(vmSpec.Disk), "0") {
	// 	keyValue := irs.KeyValue {
	// 		Key:   "Notice",
	// 		Value: "Specify 'RootDiskType' and 'RootDiskSize' when VM Creation to Boot from the Attached Volume!!",	
	// 	}
	// 	vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList, keyValue)
	// }

	return &vmSpecInfo
}
