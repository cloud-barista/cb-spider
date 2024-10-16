package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"strconv"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VMSpec = "VMSPEC"
)

type AzureVmSpecHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *armcompute.VirtualMachineSizesClient
}

func setterVmSpec(region string, vmSpec *armcompute.VirtualMachineSize) *irs.VMSpecInfo {
	vmSpecInfo := &irs.VMSpecInfo{
		Region:       region,
		Name:         *vmSpec.Name,
		VCpu:         irs.VCpuInfo{Count: strconv.FormatInt(int64(*vmSpec.NumberOfCores), 10)},
		Mem:          strconv.FormatInt(int64(*vmSpec.MemoryInMB), 10),
		Gpu:          nil,
		KeyValueList: nil,
	}
	return vmSpecInfo
}

func (vmSpecHandler *AzureVmSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, VMSpec, "ListVMSpec()")

	start := call.Start()

	var vmSpecList []*armcompute.VirtualMachineSize

	pager := vmSpecHandler.Client.NewListPager(vmSpecHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(vmSpecHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}

		for _, vmSpec := range page.Value {
			vmSpecList = append(vmSpecList, vmSpec)
		}
	}

	LoggingInfo(hiscallInfo, start)

	vmSpecInfoList := make([]*irs.VMSpecInfo, len(vmSpecList))
	for i, spec := range vmSpecList {
		vmSpecInfoList[i] = setterVmSpec(vmSpecHandler.Region.Region, spec)
	}
	return vmSpecInfoList, nil
}

func (vmSpecHandler *AzureVmSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, Name, "GetVMSpec()")

	start := call.Start()

	var vmSpecList []*armcompute.VirtualMachineSize

	pager := vmSpecHandler.Client.NewListPager(vmSpecHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(vmSpecHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.VMSpecInfo{}, getErr
		}

		for _, vmSpec := range page.Value {
			vmSpecList = append(vmSpecList, vmSpec)
		}
	}

	for _, spec := range vmSpecList {
		if Name == *spec.Name {
			LoggingInfo(hiscallInfo, start)
			vmSpecInfo := setterVmSpec(vmSpecHandler.Region.Region, spec)
			return *vmSpecInfo, nil
		}
	}
	getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = Not Exist"))
	cblogger.Error(getErr.Error())
	LoggingError(hiscallInfo, getErr)
	return irs.VMSpecInfo{}, getErr
}

func (vmSpecHandler *AzureVmSpecHandler) ListOrgVMSpec() (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, VMSpec, "ListOrgVMSpec()")

	start := call.Start()

	var vmSpecList []*armcompute.VirtualMachineSize

	pager := vmSpecHandler.Client.NewListPager(vmSpecHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(vmSpecHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
		}

		for _, vmSpec := range page.Value {
			vmSpecList = append(vmSpecList, vmSpec)
		}
	}

	LoggingInfo(hiscallInfo, start)

	var jsonResult struct {
		Result []*armcompute.VirtualMachineSize `json:"list"`
	}
	jsonResult.Result = vmSpecList
	jsonBytes, err := json.Marshal(jsonResult)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

func (vmSpecHandler *AzureVmSpecHandler) GetOrgVMSpec(Name string) (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, Name, "GetOrgVMSpec()")

	start := call.Start()

	var vmSpecList []*armcompute.VirtualMachineSize

	pager := vmSpecHandler.Client.NewListPager(vmSpecHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(vmSpecHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
		}

		for _, vmSpec := range page.Value {
			vmSpecList = append(vmSpecList, vmSpec)
		}
	}

	LoggingInfo(hiscallInfo, start)

	for _, spec := range vmSpecList {
		if Name == *spec.Name {
			jsonBytes, err := json.Marshal(spec)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return "", getErr
			}

			jsonString := string(jsonBytes)
			return jsonString, nil
		}
	}
	getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = Not Exist"))
	cblogger.Error(getErr.Error())
	LoggingError(hiscallInfo, getErr)
	return "", getErr
}
