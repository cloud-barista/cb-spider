package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strconv"
)

type AzureVmSpecHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *compute.VirtualMachineSizesClient
}

func (vmSpecHandler *AzureVmSpecHandler) setterVmSpec(vmSpec compute.VirtualMachineSize) *irs.VMSpecInfo {
	vmSpecInfo := &irs.VMSpecInfo{
		Region:       vmSpecHandler.Region.ResourceGroup,
		Name:         *vmSpec.Name,
		VCpu:         irs.VCpuInfo{Count: strconv.FormatInt(int64(*vmSpec.NumberOfCores), 10)},
		Mem:          strconv.FormatInt(int64(*vmSpec.MemoryInMB), 10),
		Gpu:          nil,
		KeyValueList: nil,
	}
	return vmSpecInfo
}

func (vmSpecHandler *AzureVmSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	result, err := vmSpecHandler.Client.List(vmSpecHandler.Ctx, vmSpecHandler.Region.Region)
	if err != nil {
		return nil, err
	}

	var vmSpecList []*irs.VMSpecInfo
	for _, spec := range *result.Value {
		vmSpecInfo := vmSpecHandler.setterVmSpec(spec)
		vmSpecList = append(vmSpecList, vmSpecInfo)
	}
	return vmSpecList, nil
}

func (vmSpecHandler *AzureVmSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	result, err := vmSpecHandler.Client.List(vmSpecHandler.Ctx, vmSpecHandler.Region.Region)
	if err != nil {
		return irs.VMSpecInfo{}, err
	}

	for _, spec := range *result.Value {
		if Name == *spec.Name {
			vmSpecInfo := vmSpecHandler.setterVmSpec(spec)
			return *vmSpecInfo, nil
		}
	}

	return irs.VMSpecInfo{}, nil
}

func (vmSpecHandler *AzureVmSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	result, err := vmSpecHandler.Client.List(vmSpecHandler.Ctx, vmSpecHandler.Region.Region)
	if err != nil {
		return "", err
	}

	var vmSpecList []*irs.VMSpecInfo
	for _, spec := range *result.Value {
		vmSpecInfo := vmSpecHandler.setterVmSpec(spec)
		vmSpecList = append(vmSpecList, vmSpecInfo)
	}

	jsonBytes, err := json.Marshal(vmSpecList)
	if err != nil {
		panic(err)
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}

func (vmSpecHandler *AzureVmSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	result, err := vmSpecHandler.Client.List(vmSpecHandler.Ctx, vmSpecHandler.Region.Region)
	if err != nil {
		return "", err
	}

	for _, spec := range *result.Value {
		if Name == *spec.Name {
			vmSpecInfo := vmSpecHandler.setterVmSpec(spec)

			jsonBytes, err := json.Marshal(vmSpecInfo)
			if err != nil {
				cblogger.Error(fmt.Sprintf("failed to get VM spec, err : %s", err))
				return "", err
			}

			jsonString := string(jsonBytes)
			return jsonString, nil
		}
	}

	cblogger.Error(fmt.Sprintf("failed to get VM spec, err : %s", err))
	notFoundErr := errors.New("failed to get VM spec")
	return "", notFoundErr
}
