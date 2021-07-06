package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"strings"
)

type IbmVmSpecHandler struct {
	CredentialInfo       idrv.CredentialInfo
	Region               idrv.RegionInfo
	ProductPackageClient *services.Product_Package
}

func (vmHandler *IbmVmSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VMSPEC, "VMSpec", "ListVMSpec()")

	productFilter := filter.Path("keyName").Eq(productName).Build()
	presetMask := "mask[prices[item],locations,locationCount,id,name,description,computeGroup,categories,keyName]"
	products, err := vmHandler.ProductPackageClient.Filter(productFilter).Mask("id").GetAllObjects()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	start := call.Start()
	allPresets, err := vmHandler.ProductPackageClient.Id(*products[0].Id).Mask(presetMask).GetActivePresets()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	var availablePresets []datatypes.Product_Package_Preset
	for _, preset := range allPresets {
		// all Region avail
		if preset.LocationCount == nil || *preset.LocationCount == uint(0) {
			availablePresets = append(availablePresets, preset)
		} else {
			// Region check
			for _, location := range preset.Locations {
				if *location.Name == Region {
					availablePresets = append(availablePresets, preset)
					break
				}
			}
		}
	}
	var vmSpecInfos []*irs.VMSpecInfo

	for _, availablePreset := range availablePresets {
		spec, err := GetVmSpecFromPreset(availablePreset, Region)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		vmSpecInfos = append(vmSpecInfos, &spec)
	}
	LoggingInfo(hiscallInfo, start)
	return vmSpecInfos, nil
}

func (vmHandler *IbmVmSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VMSPEC, Name, "GetVMSpec()")

	nameFilter := filter.Path("activePresets.keyName").Eq(Name).Build()
	productFilter := filter.Path("keyName").Eq(productName).Build()
	presetMask := "mask[prices[item],locations,locationCount,id,name,description,computeGroup,categories,keyName]"
	start := call.Start()
	products, err := vmHandler.ProductPackageClient.Filter(productFilter).Mask("id").GetAllObjects()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMSpecInfo{}, err
	}
	presets, err := vmHandler.ProductPackageClient.Id(*products[0].Id).Mask(presetMask).Filter(nameFilter).GetActivePresets()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMSpecInfo{}, err
	}

	if len(presets) > 0 {
		preset := presets[0]
		// 지역확인하기.
		if preset.LocationCount == nil || *preset.LocationCount != 0 {
			for _, location := range preset.Locations {
				if *location.Name == Region {
					spec, err := GetVmSpecFromPreset(preset, Region)
					if err != nil {
						LoggingError(hiscallInfo, err)
						return irs.VMSpecInfo{}, err
					}
					LoggingInfo(hiscallInfo, start)
					return spec, nil
				} else {
					err = errors.New(fmt.Sprintf("Preset %s is Not Avail Preset on %s", Name, Region))
					LoggingError(hiscallInfo, err)
					return irs.VMSpecInfo{}, err
				}
			}
			err = errors.New(fmt.Sprintf("Not Exist %s", Name))
			LoggingError(hiscallInfo, err)
			return irs.VMSpecInfo{}, err
		} else {
			spec, err := GetVmSpecFromPreset(preset, Region)
			if err != nil {
				LoggingError(hiscallInfo, err)
				return irs.VMSpecInfo{}, err
			}
			LoggingInfo(hiscallInfo, start)
			return spec, nil
		}
	} else {
		err = errors.New(fmt.Sprintf("Not Exist %s", Name))
		LoggingError(hiscallInfo, err)
		return irs.VMSpecInfo{}, err
	}
}

// return string: json format
func (vmHandler *IbmVmSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VMSPEC, "OrgVMSpec", "ListOrgVMSpec()")
	start := call.Start()
	vmSpecs, err := vmHandler.ListVMSpec(Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("failed to get VM spec, err : %s", err))
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	jsonBytes, err := json.Marshal(vmSpecs)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return "", err
	}
	jsonString := string(jsonBytes)
	LoggingInfo(hiscallInfo, start)
	return jsonString, nil
}

// return string: json format
func (vmHandler *IbmVmSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VMSPEC, "OrgVMSpec", "GetOrgVMSpec()")
	start := call.Start()
	vmSpec, err := vmHandler.GetVMSpec(Region, Name)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("failed to get VM spec List, err : %s", err))
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	jsonBytes, err := json.Marshal(vmSpec)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return "", err
	}
	jsonString := string(jsonBytes)
	LoggingInfo(hiscallInfo, start)
	return jsonString, nil
}

func GetVmSpecFromPreset(preset datatypes.Product_Package_Preset, Region string) (irs.VMSpecInfo, error) {
	var presetGpuInfos []irs.GpuInfo
	presetMem := ""
	presetVcpu := irs.VCpuInfo{}
	for _, price := range preset.Prices {
		if strings.Contains(*price.Item.KeyName, "RAM") {
			capacity := fmt.Sprintf("%.0f%s", *price.Item.Capacity, *price.Item.Units)
			presetMem = capacity
		}
		if strings.Contains(*price.Item.KeyName, "CORE") {
			Clock := ""
			descriptionSplits := strings.Split(*price.Item.Description, " ")
			for index, v := range descriptionSplits {
				if v == "GHz" && index > 0 {
					Clock = descriptionSplits[index-1]
				}
			}
			vCpuInfo := irs.VCpuInfo{
				Count: fmt.Sprintf("%.0f", *price.Item.Capacity),
				Clock: Clock,
			}
			presetVcpu = vCpuInfo
		}
		if strings.Contains(*price.Item.KeyName, "GPU") {
			model := ""
			keyNameSplits := strings.Split(*price.Item.KeyName, "_")
			for index, v := range keyNameSplits {
				if v == "GPU" && index > 0 {
					model = keyNameSplits[index-1]
				}
			}
			gpuInfo := irs.GpuInfo{
				Count: fmt.Sprintf("%.0f", *price.Item.Capacity),
				Model: model,
			}
			presetGpuInfos = append(presetGpuInfos, gpuInfo)
		}

	}
	spec := irs.VMSpecInfo{
		Region: Region,
		Name:   *preset.KeyName,
		VCpu:   presetVcpu,
		Mem:    presetMem,
		Gpu:    presetGpuInfos,
	}
	return spec, nil
}
