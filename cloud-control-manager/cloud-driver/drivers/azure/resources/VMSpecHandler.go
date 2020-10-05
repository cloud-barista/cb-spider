package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"

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
	Client *compute.VirtualMachineSizesClient
}

func setterVmSpec(region string, vmSpec compute.VirtualMachineSize) *irs.VMSpecInfo {
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

func (vmSpecHandler *AzureVmSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	// log HisCall
	cblogger.Info("Call Azure ListVMSpec()")
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, VMSpec, "ListVMSpec()")

	start := call.Start()
	result, err := vmSpecHandler.Client.List(vmSpecHandler.Ctx, Region)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return nil, err
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	vmSpecList := make([]*irs.VMSpecInfo, len(*result.Value))
	for i, spec := range *result.Value {
		vmSpecList[i] = setterVmSpec(Region, spec)
	}
	return vmSpecList, nil
}

func (vmSpecHandler *AzureVmSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	// log HisCall
	cblogger.Info("Call Azure GetVMSpec()")
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, Name, "GetVMSpec()")

	start := call.Start()
	result, err := vmSpecHandler.Client.List(vmSpecHandler.Ctx, Region)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return irs.VMSpecInfo{}, err
	}

	for _, spec := range *result.Value {
		if Name == *spec.Name {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			calllogger.Info(call.String(hiscallInfo))
			vmSpecInfo := setterVmSpec(Region, spec)
			return *vmSpecInfo, nil
		}
	}

	errMsg := fmt.Sprintf("failed to get VM spec, err : %s", err)
	getErr := errors.New(errMsg)
	cblogger.Error(getErr.Error())
	hiscallInfo.ErrorMSG = getErr.Error()
	calllogger.Info(call.String(hiscallInfo))
	return irs.VMSpecInfo{}, getErr
}

func (vmSpecHandler *AzureVmSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	// log HisCall
	cblogger.Info("Call Azure ListOrgVMSpec()")
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, VMSpec, "ListOrgVMSpec()")

	start := call.Start()
	result, err := vmSpecHandler.Client.List(vmSpecHandler.Ctx, Region)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return "", err
	}

	var jsonResult struct {
		Result []compute.VirtualMachineSize `json:"list"`
	}
	jsonResult.Result = *result.Value
	jsonBytes, err := json.Marshal(jsonResult)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return "", err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	jsonString := string(jsonBytes)
	return jsonString, nil
}

func (vmSpecHandler *AzureVmSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	// log HisCall
	cblogger.Info("Call Azure GetOrgVMSpec()")
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, Name, "GetOrgVMSpec()")

	start := call.Start()
	result, err := vmSpecHandler.Client.List(vmSpecHandler.Ctx, Region)
	if err != nil {
		return "", err
	}

	for _, spec := range *result.Value {
		if Name == *spec.Name {
			jsonBytes, err := json.Marshal(spec)
			if err != nil {
				errMsg := fmt.Sprintf("failed to get VM spec, err : %s", err)
				getErr := errors.New(errMsg)
				cblogger.Error(getErr.Error())
				hiscallInfo.ErrorMSG = getErr.Error()
				calllogger.Info(call.String(hiscallInfo))
				return "", err
			}

			hiscallInfo.ElapsedTime = call.Elapsed(start)
			calllogger.Info(call.String(hiscallInfo))

			jsonString := string(jsonBytes)
			return jsonString, nil
		}
	}

	notFoundErr := errors.New("failed to get VM spec")
	cblogger.Error(notFoundErr.Error())
	hiscallInfo.ErrorMSG = notFoundErr.Error()
	calllogger.Info(call.String(hiscallInfo))
	return "", notFoundErr
}
