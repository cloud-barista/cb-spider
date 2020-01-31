package resources

import (
	"encoding/json"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"strconv"
)

type OpenStackVMSpecHandler struct {
	Client *gophercloud.ServiceClient
}

func setterVMSpec(vmSpec flavors.Flavor) *irs.VMSpecInfo {
	vmSpecInfo := &irs.VMSpecInfo{
		//Region:       "",
		Name: vmSpec.Name,
		VCpu: irs.VCpuInfo{Count: strconv.Itoa(vmSpec.VCPUs)},
		Mem:  strconv.Itoa(vmSpec.RAM),
		//Gpu:          nil,
		//KeyValueList: nil,
	}

	return vmSpecInfo
}

func (vmSpecHandler *OpenStackVMSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	pager, err := flavors.ListDetail(vmSpecHandler.Client, flavors.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := flavors.ExtractFlavors(pager)
	if err != nil {
		return nil, err
	}

	vmSpecList := make([]*irs.VMSpecInfo, len(list))
	for i, spec := range list {
		vmSpecList[i] = setterVMSpec(spec)
	}
	return vmSpecList, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	vmSpecId, err := flavors.IDFromName(vmSpecHandler.Client, Name)
	if err != nil {
		return irs.VMSpecInfo{}, err
	}
	vmSpec, err := flavors.Get(vmSpecHandler.Client, vmSpecId).Extract()
	if err != nil {
		return irs.VMSpecInfo{}, err
	}

	vmSpecInfo := setterVMSpec(*vmSpec)
	return *vmSpecInfo, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	pager, err := flavors.ListDetail(vmSpecHandler.Client, flavors.ListOpts{}).AllPages()
	if err != nil {
		return "", err
	}
	list, err := flavors.ExtractFlavors(pager)
	if err != nil {
		return "", err
	}

	vmSpecList := make([]*irs.VMSpecInfo, len(list))
	for i, spec := range list {
		vmSpecList[i] = setterVMSpec(spec)
	}

	jsonBytes, err := json.Marshal(vmSpecList)
	if err != nil {
		panic(err)
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}

func (vmSpecHandler *OpenStackVMSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	vmSpecId, err := flavors.IDFromName(vmSpecHandler.Client, Name)
	if err != nil {
		return "", err
	}
	vmSpec, err := flavors.Get(vmSpecHandler.Client, vmSpecId).Extract()
	if err != nil {
		return "", err
	}

	vmSpecInfo := setterVMSpec(*vmSpec)

	jsonBytes, err := json.Marshal(vmSpecInfo)
	if err != nil {
		return "", err
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}
