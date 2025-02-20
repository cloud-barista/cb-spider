package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VMSpec = "VMSPEC"
)

type OpenStackVMSpecHandler struct {
	Region idrv.RegionInfo
	Client *gophercloud.ServiceClient
}

func setterVMSpec(region string, vmSpec flavors.Flavor) *irs.VMSpecInfo {
	vmSpecInfo := &irs.VMSpecInfo{
		Region: region,
		Name:   vmSpec.Name,
		VCpu:   irs.VCpuInfo{Count: strconv.Itoa(vmSpec.VCPUs), Clock: "-1"},
		Mem:    strconv.Itoa(vmSpec.RAM),
		Disk:   strconv.Itoa(vmSpec.Disk),
		Gpu:    nil,

		KeyValueList: irs.StructToKeyValueList(vmSpec),
	}

	return vmSpecInfo
}

func (vmSpecHandler *OpenStackVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, VMSpec, "ListVMSpec()")
	start := call.Start()
	pager, err := flavors.ListDetail(vmSpecHandler.Client, flavors.ListOpts{}).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	list, err := flavors.ExtractFlavors(pager)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	vmSpecList := make([]*irs.VMSpecInfo, len(list))
	for i, spec := range list {
		vmSpecList[i] = setterVMSpec(vmSpecHandler.Region.Region, spec)
	}
	return vmSpecList, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, Name, "GetVMSpec()")
	start := call.Start()
	vmSpecId, err := vmSpecHandler.getIDFromName(vmSpecHandler.Client, Name)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMSpecInfo{}, getErr
	}

	vmSpec, err := flavors.Get(vmSpecHandler.Client, vmSpecId).Extract()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMSpecInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vmSpecInfo := setterVMSpec(vmSpecHandler.Region.Region, *vmSpec)
	return *vmSpecInfo, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) ListOrgVMSpec() (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, VMSpec, "ListOrgVMSpec()")
	start := call.Start()
	pager, err := flavors.ListDetail(vmSpecHandler.Client, flavors.ListOpts{}).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	list, err := flavors.ExtractFlavors(pager)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	var jsonResult struct {
		Result []flavors.Flavor `json:"list"`
	}
	jsonResult.Result = list
	jsonBytes, err := json.Marshal(jsonResult)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)
	LoggingInfo(hiscallInfo, start)

	return jsonString, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) GetOrgVMSpec(Name string) (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, Name, "GetOrgVMSpec()")
	start := call.Start()
	vmSpecId, err := vmSpecHandler.getIDFromName(vmSpecHandler.Client, Name)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	vmSpec, err := flavors.Get(vmSpecHandler.Client, vmSpecId).Extract()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonBytes, err := json.Marshal(vmSpec)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)
	LoggingInfo(hiscallInfo, start)
	return jsonString, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) getIDFromName(serviceClient *gophercloud.ServiceClient, imageName string) (string, error) {
	pager, err := flavors.ListDetail(serviceClient, flavors.ListOpts{}).AllPages()
	if err != nil {
		return "", err
	}
	flavorList, err := flavors.ExtractFlavors(pager)
	if err != nil {
		return "", err
	}

	var flavorNameList []flavors.Flavor
	for _, flavor := range flavorList {
		if flavor.Name == imageName {
			flavorNameList = append(flavorNameList, flavor)
		}
	}

	if len(flavorNameList) > 1 {
		return "", errors.New(fmt.Sprintf("found multiple images with name %s", imageName))
	} else if len(flavorNameList) == 0 {
		return "", errors.New(fmt.Sprintf("could not found image with name %s", imageName))
	}
	return flavorNameList[0].ID, nil
}
