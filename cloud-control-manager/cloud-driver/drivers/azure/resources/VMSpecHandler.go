package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VMSpec = "VMSPEC"
)

type AzureVmSpecHandler struct {
	Region             idrv.RegionInfo
	Ctx                context.Context
	Client             *armcompute.VirtualMachineSizesClient
	ResourceSKUsClient *armcompute.ResourceSKUsClient
}

func getGpuCount(vmSize string) (ea float32, memory int) {
	// NC series
	// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/nc-family
	if strings.HasPrefix(vmSize, "nc") {
		if strings.Contains(vmSize, "h100") {
			if strings.Contains(vmSize, "nc40") || strings.Contains(vmSize, "ncc40") {
				return 1, 94
			} else if strings.Contains(vmSize, "nc80") {
				return 2, 188
			}
		} else if strings.Contains(vmSize, "t4") {
			if strings.Contains(vmSize, "nc4") {
				return 1, 16
			} else if strings.Contains(vmSize, "nc8") {
				return 1, 16
			} else if strings.Contains(vmSize, "nc16") {
				return 1, 16
			} else if strings.Contains(vmSize, "nc64") {
				return 4, 64
			}
		} else if strings.Contains(vmSize, "a100") {
			if strings.Contains(vmSize, "nc24") {
				return 1, 80
			} else if strings.Contains(vmSize, "nc48") {
				return 2, 160
			} else if strings.Contains(vmSize, "nc96") {
				return 4, 320
			}
		} else {
			re := regexp.MustCompile(`v(\d+)`)
			matches := re.FindStringSubmatch(vmSize)
			if len(matches) > 1 {
				version, _ := strconv.Atoi(matches[1])
				if version == 2 || version == 3 {
					if strings.Contains(vmSize, "nc24") {
						return 4, 64
					} else if strings.Contains(vmSize, "nc12") {
						return 2, 32
					} else if strings.Contains(vmSize, "nc6") {
						return 1, 16
					}
				}
			} else {
				if strings.Contains(vmSize, "nc24") {
					return 4, 48
				} else if strings.Contains(vmSize, "nc12") {
					return 2, 24
				} else if strings.Contains(vmSize, "nc6") {
					return 1, 12
				}
			}
		}
	}

	// ND series
	// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/nd-family
	if strings.HasPrefix(vmSize, "nd") {
		if strings.Contains(vmSize, "h100") {
			if strings.Contains(vmSize, "nd96") {
				return 8, 640
			}
		} else if strings.Contains(vmSize, "h200") {
			if strings.Contains(vmSize, "nd96") {
				return 8, 1128
			}
		} else if strings.Contains(vmSize, "mi300x") {
			if strings.Contains(vmSize, "nd96") {
				return 8, 1535
			}
		} else if strings.Contains(vmSize, "a100") {
			if strings.Contains(vmSize, "asr") {
				if strings.Contains(vmSize, "nd96") {
					return 8, 320
				}
			} else if strings.Contains(vmSize, "amsr") {
				if strings.Contains(vmSize, "nd96") {
					return 8, 80
				}
			}
		} else {
			re := regexp.MustCompile(`v(\d+)`)
			matches := re.FindStringSubmatch(vmSize)
			if len(matches) > 1 {
				version, _ := strconv.Atoi(matches[1])
				if version == 2 {
					if strings.Contains(vmSize, "nd40") {
						// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/ndv2-series?tabs=sizeaccelerators#sizes-in-series
						return 8, 256
					}
				} else if version == 3 {
					if strings.Contains(vmSize, "nd40") {
						return 8, 256 // V100 GPU x8
					}
				}
			} else {
				if strings.Contains(vmSize, "nd24") {
					return 4, 96
				} else if strings.Contains(vmSize, "nd12") {
					return 2, 48
				} else if strings.Contains(vmSize, "nd6") {
					return 1, 24
				}
			}
		}
	}

	// NG series
	// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/ng-family
	if strings.HasPrefix(vmSize, "ng") {
		if strings.Contains(vmSize, "v620") {
			if strings.Contains(vmSize, "ng8") {
				return 0.25, 8
			} else if strings.Contains(vmSize, "ng16") {
				return 0.5, 16
			} else if strings.Contains(vmSize, "ng32") {
				return 1, 32
			}
		}
	}

	// NV series
	// https://learn.microsoft.com/ko-kr/azure/virtual-machines/sizes/gpu-accelerated/nv-family
	if strings.HasPrefix(vmSize, "nv") {
		// A10 GPU series
		if strings.Contains(vmSize, "a10") {
			if strings.Contains(vmSize, "nv6") {
				// original quantity: 1/6 = 0.167
				return 0.167, 8
			} else if strings.Contains(vmSize, "nv12") {
				// original quantity: 1/3 = 0.333
				return 0.333, 8
			} else if strings.Contains(vmSize, "nv18") {
				return 0.5, 12
			} else if strings.Contains(vmSize, "nv36") {
				return 1, 24
			} else if strings.Contains(vmSize, "nv72") {
				return 2, 48
			}
			return -1, -1
		}

		// V710 GPU series
		if strings.Contains(vmSize, "v710") {
			if strings.Contains(vmSize, "nv4") {
				// original quantity: 1/6 = 0.167
				return 0.167, 4
			} else if strings.Contains(vmSize, "nv8") {
				// original quantity: 1/3 = 0.333
				return 0.333, 8
			} else if strings.Contains(vmSize, "nv12") {
				return 0.5, 12
			} else if strings.Contains(vmSize, "nv24") {
				return 1, 24
			} else if strings.Contains(vmSize, "nv28") {
				return 1, 24
			}
			return -1, -1
		}

		// V2 series
		if strings.Contains(vmSize, "v2") {
			if strings.Contains(vmSize, "nv6") {
				return 1, 8 // Tesla M60 GPU
			} else if strings.Contains(vmSize, "nv12") {
				return 2, 16 // Tesla M60 GPU x2
			} else if strings.Contains(vmSize, "nv24") {
				return 4, 32 // Tesla M60 GPU x4
			}
			return -1, -1
		}

		// V3 series
		if strings.Contains(vmSize, "v3") {
			if strings.Contains(vmSize, "nv12") {
				return 1, 8
			} else if strings.Contains(vmSize, "nv24") {
				return 2, 16
			} else if strings.Contains(vmSize, "nv48") {
				return 4, 32
			}
			return -1, -1
		}

		// V4 series
		if strings.Contains(vmSize, "v4") {
			// checak if "as" is in the vmSize
			if strings.Contains(vmSize, "as") {
				if strings.Contains(vmSize, "nv8") {
					return 0.25, 4
				} else if strings.Contains(vmSize, "nv16") {
					return 0.5, 8
				} else if strings.Contains(vmSize, "nv32") {
					return 1, 16
				}
			} else {
				if strings.Contains(vmSize, "nv4") {
					return 0.125, 2
				} else if strings.Contains(vmSize, "nv8") {
					return 0.25, 4
				} else if strings.Contains(vmSize, "nv16") {
					return 0.5, 8
				} else if strings.Contains(vmSize, "nv32") {
					return 1, 16
				}
			}
			return -1, -1
		}

		// NV series no version
		if strings.Contains(vmSize, "nd24") {
			return 4, 96
		} else if strings.Contains(vmSize, "nd12") {
			return 2, 48
		} else if strings.Contains(vmSize, "nd6") {
			return 1, 24
		}
	}

	return -1, -1
}

func getGpuModel(vmSize string) string {
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
		if strings.Contains(vmSize, "v2") {
			return "NVIDIA Tesla P100"
		} else if strings.Contains(vmSize, "v3") {
			return "NVIDIA Tesla V100"
		} else {
			return "NVIDIA Tesla K80"
		}
	}

	// ND series
	if strings.Contains(vmSize, "nd") {
		if strings.Contains(vmSize, "asr") {
			return "NVIDIA A100"
		} else if strings.Contains(vmSize, "v2") {
			return "NVIDIA Tesla V100 NVLINK"
		} else if strings.Contains(vmSize, "v3") {
			return "NVIDIA Tesla V100"
		} else {
			return "NVIDIA Tesla P40"
		}
	}

	// NV series
	if strings.Contains(vmSize, "nv") {
		if strings.Contains(vmSize, "v4") {
			return "AMD Radeon Instinct MI25"
		} else if strings.Contains(vmSize, "v2") || strings.Contains(vmSize, "v3") {
			return "NVIDIA Tesla M60"
		} else {
			return "NVIDIA Tesla M60"
		}
	}

	return "NA"
}

func parseGpuInfo(vmSizeName string) *irs.GpuInfo {
	vmSizeLower := strings.ToLower(vmSizeName)
	vmSizeLower = strings.ReplaceAll(vmSizeLower, "standard_", "")

	// check if the vmSize is in a GPU series
	isGpuSeries := false
	for _, prefix := range []string{"nc", "nd", "ng", "nv"} {
		if strings.HasPrefix(vmSizeLower, prefix) {
			isGpuSeries = true
			break
		}
	}

	if !isGpuSeries {
		cblogger.Infof("VM %s is not in a GPU series", vmSizeLower)
		return nil
	}

	count, mem := getGpuCount(vmSizeLower)

	// invalid gpu info
	if count == -1 || mem == -1 {
		cblogger.Warningf("Could not determine GPU info for %s (count: %f, mem: %d)",
			vmSizeLower, count, mem)

		// set default values
		vmSizeBase := vmSizeLower

		vmSizeBase = strings.Replace(vmSizeBase, "s_v", "_v", 1)  // "s_v" -> "_v"
		vmSizeBase = strings.Replace(vmSizeBase, "sv", "v", 1)    // "sv" -> "v"
		vmSizeBase = strings.Replace(vmSizeBase, "as_v", "_v", 1) // "as_v" -> "_v"
		vmSizeBase = strings.Replace(vmSizeBase, "asv", "v", 1)   // "asv" -> "v"

		cblogger.Infof("Transformed VM name for matching: %s -> %s", vmSizeLower, vmSizeBase)

		// NV series
		if strings.HasPrefix(vmSizeLower, "nv") {
			// NVv2 series
			if strings.Contains(vmSizeBase, "v2") {
				if strings.Contains(vmSizeBase, "nv6") {
					count = 1
					mem = 8
				} else if strings.Contains(vmSizeBase, "nv12") {
					count = 2
					mem = 16
				} else if strings.Contains(vmSizeBase, "nv24") {
					count = 4
					mem = 32
				}
			}

			// NVv3 series
			if strings.Contains(vmSizeBase, "v3") {
				if strings.Contains(vmSizeBase, "nv12") {
					count = 1
					mem = 8
				} else if strings.Contains(vmSizeBase, "nv24") {
					count = 2
					mem = 16
				} else if strings.Contains(vmSizeBase, "nv48") {
					count = 4
					mem = 32
				}
			}

			// NVv4 series
			if strings.Contains(vmSizeBase, "v4") {
				if strings.Contains(vmSizeBase, "nv8") {
					count = 0.25
					mem = 4
				} else if strings.Contains(vmSizeBase, "nv16") {
					count = 0.5
					mem = 8
				} else if strings.Contains(vmSizeBase, "nv32") {
					count = 1
					mem = 16
				} else if strings.Contains(vmSizeBase, "nv4") {
					count = 0.125
					mem = 2
				}
			}
		}

		// ND series
		if strings.HasPrefix(vmSizeLower, "nd") {
			if strings.Contains(vmSizeBase, "nd40") {
				count = 8
				mem = 256
			} else if strings.Contains(vmSizeBase, "nd96") {
				count = 8
				mem = 320
			}
		}

		// case where GPU info is still not determined
		if count == -1 || mem == -1 {
			// set vm size from vmSizeLower
			reSize := regexp.MustCompile(`(\d+)`)
			sizeMatches := reSize.FindStringSubmatch(vmSizeLower)
			if len(sizeMatches) > 0 {
				sizeNum, err := strconv.Atoi(sizeMatches[1])
				if err == nil {
					if strings.HasPrefix(vmSizeLower, "nv") {
						// NV series
						if sizeNum <= 6 {
							count = 1
							mem = 8
						} else if sizeNum <= 12 {
							count = 2
							mem = 16
						} else if sizeNum <= 24 {
							count = 4
							mem = 32
						} else {
							count = 8
							mem = 64
						}
					} else if strings.HasPrefix(vmSizeLower, "nd") {
						// ND series
						if sizeNum <= 24 {
							count = 4
							mem = 96
						} else if sizeNum <= 40 {
							count = 8
							mem = 256
						} else {
							count = 8
							mem = 320
						}
					}
				}
			}

			// Fallback logic for GPU info
			if count == -1 || mem == -1 {
				if strings.HasPrefix(vmSizeLower, "nv") {
					count = 1
					mem = 8
				} else if strings.HasPrefix(vmSizeLower, "nd") {
					count = 4
					mem = 96
				} else if strings.HasPrefix(vmSizeLower, "nc") {
					count = 1
					mem = 12
				}
			}

			if count == -1 || mem == -1 {
				cblogger.Errorf("Failed to determine GPU info for %s even with fallback logic", vmSizeLower)
			} else {
				cblogger.Infof("Applied estimated GPU values for %s: count=%f, mem=%d",
					vmSizeLower, count, mem)
			}
		}
	}

	countStr := formatGpuCountValue(count)
	modelFullName := getGpuModel(vmSizeLower)

	var mfr = "NA"
	var model = "NA"
	if strings.HasPrefix(modelFullName, "NVIDIA") {
		mfr = "NVIDIA"
		model, _ = strings.CutPrefix(modelFullName, "NVIDIA ")
	} else if strings.HasPrefix(modelFullName, "AMD") {
		mfr = "AMD"
		model, _ = strings.CutPrefix(modelFullName, "AMD ")
	}

	// GPU Memroy Size
	var memPerGpu string
	if count <= 0 || mem <= 0 {
		memPerGpu = "-1"
	} else {
		memPerGpu = fmt.Sprintf("%d", int64(float64(mem)/float64(count)))
	}

	gpuInfo := &irs.GpuInfo{
		Mfr:            mfr,
		Model:          model,
		MemSizeGB:      memPerGpu,
		Count:          countStr,
		TotalMemSizeGB: fmt.Sprintf("%d", mem),
	}

	return gpuInfo
}

func formatGpuCountValue(value float32) string {
	// Check if integer
	if value == float32(int32(value)) {
		return fmt.Sprintf("%d", int32(value))
	}
	return fmt.Sprintf("%.3f", value)
}

func setterVmSpec(region string, vmSpec *armcompute.VirtualMachineSize, resourceSKU *armcompute.ResourceSKU) *irs.VMSpecInfo {
	var keyValueList []irs.KeyValue

	if vmSpec.Name == nil {
		return nil
	}

	// Add basic VM spec info to KeyValueList
	keyValueList = irs.StructToKeyValueList(vmSpec)

	// Add ResourceSKU capabilities if available
	if resourceSKU != nil && resourceSKU.Capabilities != nil {
		for _, capability := range resourceSKU.Capabilities {
			if capability.Name != nil && capability.Value != nil {
				keyValueList = append(keyValueList, irs.KeyValue{
					Key:   *capability.Name,
					Value: *capability.Value,
				})
			}
		}

		// Add LocationInfo if available
		if resourceSKU.LocationInfo != nil {
			for i, locationInfo := range resourceSKU.LocationInfo {
				if locationInfo.Location != nil {
					keyValueList = append(keyValueList, irs.KeyValue{
						Key:   fmt.Sprintf("LocationInfo_%d_Location", i),
						Value: *locationInfo.Location,
					})
				}
				if locationInfo.Zones != nil {
					for j, zone := range locationInfo.Zones {
						keyValueList = append(keyValueList, irs.KeyValue{
							Key:   fmt.Sprintf("LocationInfo_%d_Zone_%d", i, j),
							Value: *zone,
						})
					}
				}
				if locationInfo.ZoneDetails != nil {
					for j, zoneDetail := range locationInfo.ZoneDetails {
						if zoneDetail.Name != nil {
							for k, zoneName := range zoneDetail.Name {
								keyValueList = append(keyValueList, irs.KeyValue{
									Key:   fmt.Sprintf("LocationInfo_%d_ZoneDetail_%d_Name_%d", i, j, k),
									Value: *zoneName,
								})
							}
						}
						if zoneDetail.Capabilities != nil {
							for k, capability := range zoneDetail.Capabilities {
								if capability.Name != nil && capability.Value != nil {
									keyValueList = append(keyValueList, irs.KeyValue{
										Key:   fmt.Sprintf("LocationInfo_%d_ZoneDetail_%d_Capability_%d_%s", i, j, k, *capability.Name),
										Value: *capability.Value,
									})
								}
							}
						}
					}
				}
			}
		}

		// Add family, tier, size, and resourceType if available
		if resourceSKU.Family != nil {
			keyValueList = append(keyValueList, irs.KeyValue{
				Key:   "Family",
				Value: *resourceSKU.Family,
			})
		}
		if resourceSKU.Tier != nil {
			keyValueList = append(keyValueList, irs.KeyValue{
				Key:   "Tier",
				Value: *resourceSKU.Tier,
			})
		}
		if resourceSKU.Size != nil {
			keyValueList = append(keyValueList, irs.KeyValue{
				Key:   "Size",
				Value: *resourceSKU.Size,
			})
		}
		if resourceSKU.ResourceType != nil {
			keyValueList = append(keyValueList, irs.KeyValue{
				Key:   "ResourceType",
				Value: *resourceSKU.ResourceType,
			})
		}
	}

	gpuInfoList := make([]irs.GpuInfo, 0)
	gpuInfo := parseGpuInfo(*vmSpec.Name)
	if gpuInfo != nil {
		gpuInfoList = append(gpuInfoList, *gpuInfo)
	}

	vmSpecInfo := &irs.VMSpecInfo{
		Region:     region,
		Name:       *vmSpec.Name,
		VCpu:       irs.VCpuInfo{Count: strconv.FormatInt(int64(*vmSpec.NumberOfCores), 10), ClockGHz: "-1"},
		MemSizeMiB: irs.ConvertMBToMiBInt64(int64(*vmSpec.MemoryInMB)), // MB -> MiB
		// ref) https://learn.microsoft.com/en-us/azure/virtual-machines/sizes/gpu-accelerated/ncast4v3-series
		DiskSizeGB:   irs.ConvertMiBToGBInt64(int64(*vmSpec.ResourceDiskSizeInMB)), // MiB(real) -> GB
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

	// Get ResourceSKU information for enhanced capabilities
	var skuMap map[string]*armcompute.ResourceSKU
	if vmSpecHandler.ResourceSKUsClient != nil {
		skuMap = make(map[string]*armcompute.ResourceSKU)

		skuPager := vmSpecHandler.ResourceSKUsClient.NewListPager(&armcompute.ResourceSKUsClientListOptions{
			Filter: toStrPtr(fmt.Sprintf("location eq '%s'", vmSpecHandler.Region.Region)),
		})

		for skuPager.More() {
			skuPage, err := skuPager.NextPage(vmSpecHandler.Ctx)
			if err != nil {
				cblogger.Warnf("Failed to get ResourceSKU information: %s", err)
				break
			}

			for _, sku := range skuPage.Value {
				if sku.Name != nil && sku.ResourceType != nil && *sku.ResourceType == "virtualMachines" {
					skuMap[*sku.Name] = sku
				}
			}
		}
	}

	LoggingInfo(hiscallInfo, start)

	vmSpecInfoList := make([]*irs.VMSpecInfo, len(vmSpecList))
	for i, spec := range vmSpecList {
		var resourceSKU *armcompute.ResourceSKU
		if spec.Name != nil && skuMap != nil {
			resourceSKU = skuMap[*spec.Name]
		}
		vmSpecInfoList[i] = setterVmSpec(vmSpecHandler.Region.Region, spec, resourceSKU)
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

	// Get ResourceSKU information for the specific VM
	var resourceSKU *armcompute.ResourceSKU
	if vmSpecHandler.ResourceSKUsClient != nil {
		skuPager := vmSpecHandler.ResourceSKUsClient.NewListPager(&armcompute.ResourceSKUsClientListOptions{
			Filter: toStrPtr(fmt.Sprintf("location eq '%s' and name eq '%s'", vmSpecHandler.Region.Region, Name)),
		})

		for skuPager.More() {
			skuPage, err := skuPager.NextPage(vmSpecHandler.Ctx)
			if err != nil {
				cblogger.Warnf("Failed to get ResourceSKU information for %s: %s", Name, err)
				break
			}

			for _, sku := range skuPage.Value {
				if sku.Name != nil && *sku.Name == Name && sku.ResourceType != nil && *sku.ResourceType == "virtualMachines" {
					resourceSKU = sku
					break
				}
			}
			if resourceSKU != nil {
				break
			}
		}
	}

	for _, spec := range vmSpecList {
		if Name == *spec.Name {
			LoggingInfo(hiscallInfo, start)
			vmSpecInfo := setterVmSpec(vmSpecHandler.Region.Region, spec, resourceSKU)
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
