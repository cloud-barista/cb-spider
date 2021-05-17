package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VMSpec = "VMSPEC"
)

type OpenStackVMSpecHandler struct {
	Client *gophercloud.ServiceClient
}

func setterVMSpec(region string, vmSpec flavors.Flavor) *irs.VMSpecInfo {
	vmSpecInfo := &irs.VMSpecInfo{
		Region:       region,
		Name:         vmSpec.Name,
		VCpu:         irs.VCpuInfo{Count: strconv.Itoa(vmSpec.VCPUs)},
		Mem:          strconv.Itoa(vmSpec.RAM),
		Gpu:          nil,
		KeyValueList: nil,
	}

	return vmSpecInfo
}

func (vmSpecHandler *OpenStackVMSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, VMSpec, "ListVMSpec()")

	start := call.Start()
	pager, err := flavors.ListDetail(vmSpecHandler.Client, flavors.ListOpts{}).AllPages()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	list, err := flavors.ExtractFlavors(pager)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	vmSpecList := make([]*irs.VMSpecInfo, len(list))
	for i, spec := range list {
		vmSpecList[i] = setterVMSpec(Region, spec)
	}
	return vmSpecList, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, Name, "GetVMSpec()")

	vmSpecId, err := vmSpecHandler.getIDFromName(vmSpecHandler.Client, Name)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMSpecInfo{}, err
	}

	start := call.Start()
	vmSpec, err := flavors.Get(vmSpecHandler.Client, vmSpecId).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMSpecInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	vmSpecInfo := setterVMSpec(Region, *vmSpec)
	return *vmSpecInfo, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, VMSpec, "ListOrgVMSpec()")

	start := call.Start()
	pager, err := flavors.ListDetail(vmSpecHandler.Client, flavors.ListOpts{}).AllPages()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return "", err
	}
	LoggingInfo(hiscallInfo, start)

	list, err := flavors.ExtractFlavors(pager)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return "", err
	}

	var jsonResult struct {
		Result []flavors.Flavor `json:"list"`
	}
	jsonResult.Result = list
	jsonBytes, err := json.Marshal(jsonResult)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return "", err
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, Name, "GetOrgVMSpec()")

	vmSpecId, err := vmSpecHandler.getIDFromName(vmSpecHandler.Client, Name)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return "", err
	}

	start := call.Start()
	vmSpec, err := flavors.Get(vmSpecHandler.Client, vmSpecId).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return "", err
	}
	LoggingInfo(hiscallInfo, start)

	jsonBytes, err := json.Marshal(vmSpec)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return "", err
	}

	jsonString := string(jsonBytes)

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
