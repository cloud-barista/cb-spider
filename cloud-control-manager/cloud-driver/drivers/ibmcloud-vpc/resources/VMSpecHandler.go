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
	"strconv"
	"strings"
)

type IbmVmSpecHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

func (vmSpecHandler *IbmVmSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, "VMSpec", "ListVMSpec()")
	start := call.Start()
	var specList []*irs.VMSpecInfo
	options := &vpcv1.ListInstanceProfilesOptions{}
	profiles, _, err := vmSpecHandler.VpcService.ListInstanceProfilesWithContext(vmSpecHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	for _, profile := range profiles.Profiles {
		vmSpecInfo, err := setVmSpecInfo(profile, vmSpecHandler.Region.Region)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List VMSpec. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		specList = append(specList, &vmSpecInfo)
	}
	LoggingInfo(hiscallInfo, start)
	return specList, nil
}
func (vmSpecHandler *IbmVmSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
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
	vmSpecInfo, err := setVmSpecInfo(profile, vmSpecHandler.Region.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMSpecInfo{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return vmSpecInfo, nil
}

func (vmSpecHandler *IbmVmSpecHandler) ListOrgVMSpec() (string, error) {
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Region, call.VMSPEC, "OrgVMSpec", "ListOrgVMSpec()")
	start := call.Start()

	var specList []*irs.VMSpecInfo
	options := &vpcv1.ListInstanceProfilesOptions{}
	profiles, _, err := vmSpecHandler.VpcService.ListInstanceProfilesWithContext(vmSpecHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	for _, profile := range profiles.Profiles {
		vmSpecInfo, err := setVmSpecInfo(profile, vmSpecHandler.Region.Region)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
		}
		specList = append(specList, &vmSpecInfo)
	}
	jsonBytes, err := json.Marshal(specList)
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
func (vmSpecHandler *IbmVmSpecHandler) GetOrgVMSpec(Name string) (string, error) {
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
	vmSpecInfo, err := setVmSpecInfo(profile, vmSpecHandler.Region.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
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

func getGpuInfo(name string) (Mfr string, count string, model string, mem string) {
	name = strings.ToLower(name)
	//https://cloud.ibm.com/docs/vpc?topic=vpc-profiles&interface=ui#gpu
	//https://www.ibm.com/kr-ko/cloud/gpu/nvidia

	// H100 GPU
	if strings.Contains(name, "gx3d-160x1792x8h100") {
		return "NVIDIA", "8", "H100", "1835008" // 1,792 GiB -> 1,792 * 1024 MB = 1835008 MB
	}

	// L40S GPU
	if strings.Contains(name, "gx2-24x120x1l40s") {
		return "NVIDIA", "1", "L40S", "122880" // 120 GiB -> 120 * 1024 MB = 122880 MB
	}
	if strings.Contains(name, "gx3-48x240x2l40s") {
		return "NVIDIA", "2", "L40S", "245760" // 240 GiB -> 240 * 1024 MB = 245760 MB
	}
	if strings.Contains(name, "gx3-24x120x1l40s") {
		return "NVIDIA", "1", "L40S", "49152"
	}
	if strings.Contains(name, "gx3-24x120x1l40s") {
		return "NVIDIA", "2", "L40S", "98304"
	}

	// L4 GPU
	if strings.Contains(name, "gx2-16x80x1l4") {
		return "NVIDIA", "1", "L4", "81920" // 80 GiB -> 80 * 1024 MB = 81920 MB
	}
	if strings.Contains(name, "gx2-32x160x2l4") {
		return "NVIDIA", "2", "L4", "163840" // 160 GiB -> 160 * 1024 MB = 163840 MB
	}
	if strings.Contains(name, "gx2-64x320x4l4") {
		return "NVIDIA", "4", "L4", "327680" // 320 GiB -> 320 * 1024 MB = 327680 MB
	}
	if strings.Contains(name, "gx3-16x80x1l4") {
		return "NVIDIA", "1", "L4", "24576"
	}
	if strings.Contains(name, "gx3-32x160x2l4") {
		return "NVIDIA", "2", "L4", "49152"
	}
	if strings.Contains(name, "gx3-64x320x4l4") {
		return "NVIDIA", "4", "L4", "98304"
	}

	// P100 GPU
	if strings.Contains(name, "gx2-8x60x1p100") {
		return "NVIDIA", "1", "P100", "61440" // 60 GiB -> 60 * 1024 MB = 61440 MB
	}
	if strings.Contains(name, "gx2-16x120x2p100") {
		return "NVIDIA", "2", "P100", "122880" // 120 GiB -> 120 * 1024 MB = 122880 MB
	}

	// T4 GPU
	if strings.Contains(name, "gx2-8x32x1t4") {
		return "NVIDIA", "1", "T4", "32768" // 32 GiB -> 32 * 1024 MB = 32768 MB
	}
	if strings.Contains(name, "gx2-16x64x2t4") {
		return "NVIDIA", "2", "T4", "65536" // 64 GiB -> 64 * 1024 MB = 65536 MB
	}

	//V100
	if strings.Contains(name, "gx2-8x64x1v100") {
		return "NVIDIA", "1", "V100", "16384" // 32 GiB -> 32 * 1024 MB = 32768 MB
	}
	if strings.Contains(name, "gx2-16x128x1v100") {
		return "NVIDIA", "1", "V100", "16384" // 64 GiB -> 64 * 1024 MB = 65536 MB
	}
	if strings.Contains(name, "gx2-16x128x2v100") {
		return "NVIDIA", "2", "V100", "32768"
	}
	if strings.Contains(name, "gx2-32x256x2v100") {
		return "NVIDIA", "2", "V100", "32768"
	}

	return "", "", "", ""
}

func setVmSpecInfo(profile vpcv1.InstanceProfile, region string) (irs.VMSpecInfo, error) {
	if profile.Name == nil {
		return irs.VMSpecInfo{}, errors.New(fmt.Sprintf("Invalid vmspec"))
	}
	vmSpecInfo := irs.VMSpecInfo{
		Region: region,
		Name:   *profile.Name,
		Disk:   "-1",
	}

	specslice := strings.Split(*profile.Name, "-")
	if len(specslice) > 1 {
		specslice2 := strings.Split(specslice[1], "x")
		if len(specslice2) > 1 {
			vmSpecInfo.VCpu = irs.VCpuInfo{Count: specslice2[0]}
			memValue, err := strconv.Atoi(specslice2[1])
			if err != nil {
				memValue = 0
			}
			memValue = memValue * 1024
			memValueString := strconv.Itoa(memValue)
			vmSpecInfo.Mem = memValueString
		}
	}

	gpuMfr, gpuCount, gpuModel, gpuMem := getGpuInfo(*profile.Name)
	vmSpecInfo.Gpu = []irs.GpuInfo{
		{
			Mfr:   gpuMfr,
			Count: gpuCount,
			Model: gpuModel,
			Mem:   gpuMem,
		},
	}
	return vmSpecInfo, nil
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
