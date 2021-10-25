package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
)

type IbmVmSpecHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

func (vmSpecHandler *IbmVmSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, "VMSpec", "ListVMSpec()")
	start := call.Start()

	var specList []*irs.VMSpecInfo
	options := &vpcv1.ListInstanceProfilesOptions{}
	profiles, _, err := vmSpecHandler.VpcService.ListInstanceProfilesWithContext(vmSpecHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMSpecList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	for _, profile := range profiles.Profiles {
		vmSpecInfo := irs.VMSpecInfo{
			Region: vmSpecHandler.Region.Region,
			Name:   *profile.Name,
		}
		specslice := strings.Split(*profile.Name, "-")
		if len(specslice) > 1 {
			specslice2 := strings.Split(specslice[1], "x")
			if len(specslice2) > 1 {
				vmSpecInfo.VCpu = irs.VCpuInfo{Count: specslice2[0]}
				vmSpecInfo.Mem = specslice2[1] + "GB"
			}
		}
		specList = append(specList, &vmSpecInfo)
	}
	LoggingInfo(hiscallInfo, start)
	return specList, nil
}
func (vmSpecHandler *IbmVmSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, Name, "GetVMSpec()")
	start := call.Start()
	if Name == "" {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = invalid Name"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMSpecInfo{}, getErr
	}

	profile, err := getRawSpec(Name, vmSpecHandler.VpcService, vmSpecHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMSpecInfo{}, getErr
	}
	vmSpecInfo := irs.VMSpecInfo{
		Region: vmSpecHandler.Region.Region,
		Name:   *profile.Name,
	}
	specslice := strings.Split(*profile.Name, "-")
	if len(specslice) > 1 {
		specslice2 := strings.Split(specslice[1], "x")
		if len(specslice2) > 1 {
			vmSpecInfo.VCpu = irs.VCpuInfo{Count: specslice2[0]}
			vmSpecInfo.Mem = specslice2[1] + "GB"
		}
	}
	LoggingInfo(hiscallInfo, start)
	return vmSpecInfo, nil
}

func (vmSpecHandler *IbmVmSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, "OrgVMSpec", "ListOrgVMSpec()")
	start := call.Start()

	var specList []*irs.VMSpecInfo
	options := &vpcv1.ListInstanceProfilesOptions{}
	profiles, _, err := vmSpecHandler.VpcService.ListInstanceProfilesWithContext(vmSpecHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpecList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	for _, profile := range profiles.Profiles {
		vmSpecInfo := irs.VMSpecInfo{
			Region: vmSpecHandler.Region.Region,
			Name:   *profile.Name,
		}
		specslice := strings.Split(*profile.Name, "-")
		if len(specslice) > 1 {
			specslice2 := strings.Split(specslice[1], "x")
			if len(specslice2) > 1 {
				vmSpecInfo.VCpu = irs.VCpuInfo{Count: specslice2[0]}
				vmSpecInfo.Mem = specslice2[1] + "GB"
			}
		}
		specList = append(specList, &vmSpecInfo)
	}
	jsonBytes, err := json.Marshal(specList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpecList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	jsonString := string(jsonBytes)
	LoggingInfo(hiscallInfo, start)
	return jsonString, nil
}
func (vmSpecHandler *IbmVmSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, "OrgVMSpec", "GetOrgVMSpec()")
	start := call.Start()
	if Name == "" {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = invalid Name"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	profile, err := getRawSpec(Name, vmSpecHandler.VpcService, vmSpecHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	vmSpecInfo := irs.VMSpecInfo{
		Region: vmSpecHandler.Region.Region,
		Name:   *profile.Name,
	}
	specSlice := strings.Split(*profile.Name, "-")
	if len(specSlice) > 1 {
		specSlice2 := strings.Split(specSlice[1], "x")
		if len(specSlice2) > 1 {
			vmSpecInfo.VCpu = irs.VCpuInfo{Count: specSlice2[0]}
			vmSpecInfo.Mem = specSlice2[1] + "GB"
		}
	}
	jsonBytes, err := json.Marshal(vmSpecInfo)
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

func getRawSpec(specName string, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.InstanceProfile, error) {
	getInstanceProfileOptions := &vpcv1.GetInstanceProfileOptions{}
	getInstanceProfileOptions.SetName(specName)
	profile, _, err := vpcService.GetInstanceProfileWithContext(ctx, getInstanceProfileOptions)
	if err != nil {
		return vpcv1.InstanceProfile{}, err
	}
	return *profile, nil
}
