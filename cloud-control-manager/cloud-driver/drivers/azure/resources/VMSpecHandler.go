package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"strconv"
	"strings"

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

func getGpuCount(vmSize string) (ea string, memory string) {
	vmSize = strings.ToLower(vmSize)

	// NC series
	// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/nc-family
	if strings.Contains(vmSize, "nc") {
		if strings.Contains(vmSize, "h100") {
			if strings.Contains(vmSize, "nc40") || strings.Contains(vmSize, "ncc40") {
				return "1", "94"
			} else if strings.Contains(vmSize, "nc80") {
				return "2", "188"
			}
		} else if strings.Contains(vmSize, "t4") {
			if strings.Contains(vmSize, "nc4") {
				return "1", "16"
			} else if strings.Contains(vmSize, "nc8") {
				return "1", "16"
			} else if strings.Contains(vmSize, "nc16") {
				return "1", "16"
			} else if strings.Contains(vmSize, "nc64") {
				return "4", "64"
			}
		} else if strings.Contains(vmSize, "a100") {
			if strings.Contains(vmSize, "nc24") {
				return "1", "80"
			} else if strings.Contains(vmSize, "nc48") {
				return "2", "160"
			} else if strings.Contains(vmSize, "nc96") {
				return "4", "320"
			}
		} else {
			if strings.Contains(vmSize, "v2") {
				if strings.Contains(vmSize, "nc24") {
					return "4", "64"
				} else if strings.Contains(vmSize, "nc12") {
					return "2", "32"
				} else if strings.Contains(vmSize, "nc6") {
					return "1", "16"
				}
			} else {
				if strings.Contains(vmSize, "nc24") {
					return "4", "48"
				} else if strings.Contains(vmSize, "nc12") {
					return "2", "24"
				} else if strings.Contains(vmSize, "nc6") {
					return "1", "12"
				}
			}
		}
	}

	// ND series
	// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/nd-family
	if strings.Contains(vmSize, "nd") {
		if strings.Contains(vmSize, "h100") {
			if strings.Contains(vmSize, "nd96") {
				return "8", "640"
			}
		} else if strings.Contains(vmSize, "h200") {
			if strings.Contains(vmSize, "nd96") {
				return "8", "1128"
			}
		} else if strings.Contains(vmSize, "mi300x") {
			if strings.Contains(vmSize, "nd96") {
				return "8", "1535"
			}
		} else if strings.Contains(vmSize, "a100") {
			if strings.Contains(vmSize, "asr") {
				if strings.Contains(vmSize, "nd96") {
					return "8", "320"
				}
			} else if strings.Contains(vmSize, "amsr") {
				if strings.Contains(vmSize, "nd96") {
					return "8", "80"
				}
			}
		} else {
			if strings.Contains(vmSize, "v2") {
				if strings.Contains(vmSize, "nd40") {
					// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/ndv2-series?tabs=sizeaccelerators#sizes-in-series
					// Dose not provide quantity
					return "", "256"
				}
			} else {
				if strings.Contains(vmSize, "nd24") {
					return "4", "96"
				} else if strings.Contains(vmSize, "nd12") {
					return "2", "48"
				} else if strings.Contains(vmSize, "nd6") {
					return "1", "24"
				}
			}
		}
	}

	// NG series
	// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/ng-family
	if strings.Contains(vmSize, "ng") {
		if strings.Contains(vmSize, "v620") {
			if strings.Contains(vmSize, "ng8") {
				return "0.25", "8"
			} else if strings.Contains(vmSize, "ng16") {
				return "0.5", "16"
			} else if strings.Contains(vmSize, "ng32") {
				return "1", "32"
			}
		}
	}

	// NV series
	// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/nv-family
	if strings.Contains(vmSize, "nv") {
		if strings.Contains(vmSize, "a10") {
			if strings.Contains(vmSize, "nv6") {
				// original quantity: 1/6 = 0.167
				return "0.167", "8"
			} else if strings.Contains(vmSize, "nv12") {
				// original quantity: 1/3 = 0.333
				return "0.333", "8"
			} else if strings.Contains(vmSize, "nv18") {
				return "0.5", "12"
			} else if strings.Contains(vmSize, "nv36") {
				return "1", "24"
			} else if strings.Contains(vmSize, "nv72") {
				return "2", "48"
			}
		} else if strings.Contains(vmSize, "v710") {
			if strings.Contains(vmSize, "nv4") {
				// original quantity: 1/6 = 0.167
				return "0.167", "4"
			} else if strings.Contains(vmSize, "nv8") {
				// original quantity: 1/3 = 0.333
				return "0.333", "8"
			} else if strings.Contains(vmSize, "nv12") {
				return "0.5", "12"
			} else if strings.Contains(vmSize, "nv24") {
				return "1", "24"
			} else if strings.Contains(vmSize, "nv28") {
				return "1", "24"
			}
		} else {
			if strings.Contains(vmSize, "v3") {
				if strings.Contains(vmSize, "nv12") {
					return "1", "8"
				} else if strings.Contains(vmSize, "nv24") {
					return "2", "16"
				} else if strings.Contains(vmSize, "nv48") {
					return "4", "32"
				}
			} else if strings.Contains(vmSize, "v4") {
				if strings.Contains(vmSize, "nv4") {
					return "0.125", "2"
				} else if strings.Contains(vmSize, "nv8") {
					return "0.25", "4"
				} else if strings.Contains(vmSize, "nv16") {
					return "0.5", "8"
				} else if strings.Contains(vmSize, "nv32") {
					return "1", "16"
				}
			} else {
				if strings.Contains(vmSize, "nd24") {
					return "4", "96"
				} else if strings.Contains(vmSize, "nd12") {
					return "2", "48"
				} else if strings.Contains(vmSize, "nd6") {
					return "1", "24"
				}
			}
		}
	}

	return "-1", "-1"
}

func getGpuModel(vmSize string) string {
	vmSize = strings.ToLower(vmSize)

	if strings.Contains(vmSize, "h100") {
		return "NVIDIA H100"
	} else if strings.Contains(vmSize, "h200") {
		return "NVIDIA H200"
	} else if strings.Contains(vmSize, "a100") {
		return "NVIDIA A100"
	} else if strings.Contains(vmSize, "t4") {
		return "NVIDIA Tesla T4"
	} else if strings.Contains(vmSize, "a10") {
		return "NVIDIA A10"
	} else if strings.Contains(vmSize, "mi300x") {
		return "AMD Instinct MI300X"
	} else if strings.Contains(vmSize, "v620") {
		return "AMD Radeon PRO V620"
	} else if strings.Contains(vmSize, "v710") {
		return "AMD Radeon PRO V710"
	}

	// NC series
	if strings.Contains(vmSize, "nc") {
		if strings.Contains(vmSize, "v3") {
			return "NVIDIA Tesla V100"
		} else if strings.Contains(vmSize, "v2") {
			return "NVIDIA Tesla P100"
		} else {
			return "NVIDIA Tesla K80"
		}
	}

	// ND series
	if strings.Contains(vmSize, "nd") {
		if strings.Contains(vmSize, "v2") {
			return "NVIDIA Tesla V100 NVLINK"
		} else {
			return "NVIDIA Tesla P40"
		}
	}

	// NV series
	if strings.Contains(vmSize, "nv") {
		if strings.Contains(vmSize, "v4") {
			return "AMD Radeon Instinct MI25"
		} else {
			return "NVIDIA Tesla M60"
		}
	}

	return ""
}

func parseGpuInfo(vmSizeName string) *irs.GpuInfo {
	vmSizeLower := strings.ToLower(vmSizeName)

	vmSizeLower = strings.ReplaceAll(vmSizeLower, "standard", "")

	// Check if it's a GPU series
	if !strings.Contains(vmSizeLower, "nc") &&
		!strings.Contains(vmSizeLower, "nd") &&
		!strings.Contains(vmSizeLower, "ng") &&
		!strings.Contains(vmSizeLower, "nv") {
		return nil
	}

	count, mem := getGpuCount(vmSizeLower)
	modelFullName := getGpuModel(vmSizeLower)
	var mfr string
	var model string
	if strings.HasPrefix(modelFullName, "NVIDIA") {
		mfr = "NVIDIA"
		model, _ = strings.CutPrefix(modelFullName, "NVIDIA ")
	} else if strings.HasPrefix(modelFullName, "AMD") {
		mfr = "AMD"
		model, _ = strings.CutPrefix(modelFullName, "AMD ")
	}

	return &irs.GpuInfo{
		Count: count,
		Mem:   mem,
		Mfr:   mfr,
		Model: model,
	}
}

func setterVmSpec(region string, vmSpec *armcompute.VirtualMachineSize) *irs.VMSpecInfo {
	var keyValueList []irs.KeyValue

	if vmSpec.Name == nil {
		return nil
	}

	keyValueList = append(keyValueList,
		irs.KeyValue{
			Key:   "MaxDataDiskCount",
			Value: strconv.FormatInt(int64(*vmSpec.MaxDataDiskCount), 10),
		},
		irs.KeyValue{
			Key:   "ResourceDiskSizeInMB",
			Value: strconv.FormatInt(int64(*vmSpec.ResourceDiskSizeInMB), 10),
		},
		irs.KeyValue{
			Key:   "OSDiskSizeInMB",
			Value: strconv.FormatInt(int64(*vmSpec.OSDiskSizeInMB), 10),
		},
	)

	gpuInfoList := make([]irs.GpuInfo, 0)
	gpuInfo := parseGpuInfo(*vmSpec.Name)
	if gpuInfo != nil {
		gpuInfoList = append(gpuInfoList, *gpuInfo)
	}

	vmSpecInfo := &irs.VMSpecInfo{
		Region:       region,
		Name:         *vmSpec.Name,
		VCpu:         irs.VCpuInfo{Count: strconv.FormatInt(int64(*vmSpec.NumberOfCores), 10)},
		Mem:          strconv.FormatInt(int64(*vmSpec.MemoryInMB), 10),
		Disk:         strconv.FormatInt(int64(*vmSpec.OSDiskSizeInMB/1024), 10),
		Gpu:          gpuInfoList,
		KeyValueList: keyValueList,
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
