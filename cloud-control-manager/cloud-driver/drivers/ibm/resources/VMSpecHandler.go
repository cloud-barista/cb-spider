package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
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

	for {
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
		nextStr, _ := getSpecNextHref(profiles.Next)
		if nextStr != "" {
			options = &vpcv1.ListInstanceProfilesOptions{
				Start: core.StringPtr(nextStr),
			}
			profiles, _, err = vmSpecHandler.VpcService.ListInstanceProfilesWithContext(vmSpecHandler.Ctx, options)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to List VMSpec. err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
			}
		} else {
			break
		}
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
	for {
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
		nextStr, _ := getSpecNextHref(profiles.Next)
		if nextStr != "" {
			options = &vpcv1.ListInstanceProfilesOptions{
				Start: core.StringPtr(nextStr),
			}
			profiles, _, err = vmSpecHandler.VpcService.ListInstanceProfilesWithContext(vmSpecHandler.Ctx, options)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return "", getErr
			}
		} else {
			break
		}
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
	if !strings.HasPrefix(name, "gx") {
		return "NA"
	}
	// Determine manufacturer from GPU model suffix (e.g., gaudi3 → Intel, mi300x → AMD)
	model := strings.ToLower(getGpuModel(name))
	if strings.HasPrefix(model, "gaudi") {
		return "Intel"
	}
	if strings.HasPrefix(model, "mi") {
		return "AMD"
	}
	return "NVIDIA"
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

func getGpuInfo(name string) (string, string, string) {
	mfr := getGpuMfr(name)
	if mfr == "NA" {
		return mfr, "-1", "NA"
	}

	count := getGpuCount(name)
	model := getGpuModel(name)

	return mfr, count, model
}

func setVmSpecInfo(profile vpcv1.InstanceProfile, region string) (irs.VMSpecInfo, error) {
	vmSpecInfo, err := setVmSpecInfoWithVMSpecName(*profile.Name, region)
	if err != nil {
		return irs.VMSpecInfo{}, err
	}

	vmSpecInfo.KeyValueList = irs.StructToKeyValueList(profile)

	return vmSpecInfo, nil
}

func setVmSpecInfoWithVMSpecName(Name string, region string) (irs.VMSpecInfo, error) {
	if Name == "" {
		return irs.VMSpecInfo{}, errors.New(fmt.Sprintf("Invalid vmspec"))
	}
	vmSpecInfo := irs.VMSpecInfo{
		Region:     region,
		Name:       Name,
		DiskSizeGB: "-1",
	}

	// Extract vCPU and memory using regex pattern: {digits}x{digits}
	// Supports both dot (bx2.4x16) and hyphen (bx2-4x16) formats
	// Examples: bx2.4x16, bx2-4x16, bx2.metal.96x384, bx3d.128x640
	re := regexp.MustCompile(`(\d+)x(\d+)`)
	matches := re.FindStringSubmatch(Name)

	if len(matches) >= 3 { // matches[0]=full match, matches[1]=vCPU, matches[2]=memory
		vmSpecInfo.VCpu = irs.VCpuInfo{Count: matches[1], ClockGHz: "-1"}
		memValue, err := strconv.Atoi(matches[2])
		if err != nil {
			memValue = 0
		}
		vmSpecInfo.MemSizeMiB = strconv.Itoa(memValue * 1024)
	}

	vmSpecInfo.Gpu = []irs.GpuInfo{}
	if strings.HasPrefix(Name, "gx") {
		gpuMfr, gpuCount, gpuModel := getGpuInfo(Name)
		vmSpecInfo.Gpu = []irs.GpuInfo{
			{
				Mfr:            gpuMfr,
				Count:          gpuCount,
				Model:          gpuModel,
				MemSizeGB:      "-1", // Memory size in SpecName is main memory size, not GPU memory size
				TotalMemSizeGB: "-1",
			},
		}
	}

	return vmSpecInfo, nil
}

func getSpecNextHref(next *vpcv1.PageLink) (string, error) {
	if next != nil {
		href := *next.Href
		u, err := url.Parse(href)
		if err != nil {
			return "", err
		}
		paramMap, _ := url.ParseQuery(u.RawQuery)
		if paramMap != nil {
			if safe := paramMap["start"]; len(safe) > 0 {
				return safe[0], nil
			}
		}
	}
	return "", errors.New("NOT NEXT")
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
