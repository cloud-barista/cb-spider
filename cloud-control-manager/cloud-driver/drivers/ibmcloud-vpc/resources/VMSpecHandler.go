package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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

func getGpuMfr(name string) string {
	if strings.HasPrefix(name, "gx2") || strings.HasPrefix(name, "gx3") {
		return "NVIDIA"
	}
	return "NA"
}

func getGpuCount(name string) string {
	splits := strings.Split(name, "-")
	if len(splits) < 2 {
		return "-1"
	}

	specDetails := strings.Split(splits[len(splits)-1], "x")
	if len(specDetails) < 3 {
		return "-1"
	}

	gpuDetails := specDetails[len(specDetails)-1]
	for i, char := range gpuDetails {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' {
			return gpuDetails[:i]
		}
	}
	return "-1"
}

func getGpuModel(name string) string {
	splits := strings.Split(name, "-")
	if len(splits) < 2 {
		return "NA"
	}

	specDetails := strings.Split(splits[len(splits)-1], "x")
	if len(specDetails) < 3 {
		return "NA"
	}

	gpuDetails := specDetails[len(specDetails)-1]
	for i, char := range gpuDetails {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' {
			return strings.ToUpper(gpuDetails[i:])
		}
	}
	return "NA"
}

func getGpuMem(name string) string {
	splits := strings.Split(name, "-")
	if len(splits) < 2 {
		return "-1"
	}

	specDetails := strings.Split(splits[len(splits)-1], "x")
	if len(specDetails) < 2 {
		return "-1"
	}

	memGiB, err := strconv.Atoi(specDetails[1])
	if err != nil {
		return "-1"
	}
	return strconv.Itoa(memGiB * 1024) // GiB -> MiB
}

func getGpuInfo(name string) (string, string, string, string) {
	//check NVIDIA gpu
	mfr := getGpuMfr(name)
	if mfr == "NA" {
		return mfr, "-1", "NA", "-1"
	}

	count := getGpuCount(name)
	model := getGpuModel(name)
	mem := getGpuMem(name)

	return mfr, count, model, mem
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
			vmSpecInfo.VCpu = irs.VCpuInfo{Count: specslice2[0], Clock: "-1"}
			memValue, err := strconv.Atoi(specslice2[1])
			if err != nil {
				memValue = 0
			}
			memValue = memValue * 1024
			memValueString := strconv.Itoa(memValue)
			vmSpecInfo.Mem = memValueString
		}
	}

	vmSpecInfo.Gpu = []irs.GpuInfo{}
	if strings.HasPrefix(*profile.Name, "gx") {
		gpuMfr, gpuCount, gpuModel, gpuMem := getGpuInfo(*profile.Name)
		vmSpecInfo.Gpu = []irs.GpuInfo{
			{
				Mfr:   gpuMfr,
				Count: gpuCount,
				Model: gpuModel,
				Mem:   gpuMem,
			},
		}
	}

	vmSpecInfo.KeyValueList = getVMSpecKeyValueList(profile)

	return vmSpecInfo, nil
}

func getVMSpecKeyValueList(profile vpcv1.InstanceProfile) []irs.KeyValue {
	var keyValueList []irs.KeyValue

	// Convert to RawJson String
	toRawJSON := func(v interface{}) string {
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes)
	}

	if profile.Bandwidth != nil {
		kv := irs.KeyValue{
			Key:   "Bandwidth",
			Value: toRawJSON(profile.Bandwidth),
		}
		keyValueList = append(keyValueList, kv)
	}

	if profile.Family != nil {
		kv := irs.KeyValue{
			Key:   "Family",
			Value: *profile.Family,
		}
		keyValueList = append(keyValueList, kv)
	}

	if profile.PortSpeed != nil {
		kv := irs.KeyValue{
			Key:   "PortSpeed",
			Value: toRawJSON(profile.PortSpeed),
		}
		keyValueList = append(keyValueList, kv)
	}

	if profile.OsArchitecture != nil {
		kv := irs.KeyValue{
			Key:   "OsArchitecture",
			Value: toRawJSON(profile.OsArchitecture),
		}
		keyValueList = append(keyValueList, kv)
	}

	if profile.VcpuArchitecture != nil {
		kv := irs.KeyValue{
			Key:   "VcpuArchitecture",
			Value: toRawJSON(profile.VcpuArchitecture),
		}
		keyValueList = append(keyValueList, kv)
	}

	if len(profile.Disks) > 0 {
		diskInfo := make([]string, 0, len(profile.Disks))
		for _, disk := range profile.Disks {
			diskInfo = append(diskInfo, toRawJSON(disk))
		}
		diskJSON := "[" + strings.Join(diskInfo, ", ") + "]"
		kv := irs.KeyValue{
			Key:   "Disks",
			Value: diskJSON,
		}
		keyValueList = append(keyValueList, kv)
	}

	if profile.Href != nil {
		kv := irs.KeyValue{
			Key:   "Href",
			Value: *profile.Href,
		}
		keyValueList = append(keyValueList, kv)
	}

	return keyValueList
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
