package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	bssopenapi "github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaPriceInfoHandler struct {
	BssClient *bssopenapi.Client
	EcsClient *ecs.Client
}

type AliPriceInfo struct {
	Data AliData `json:"Data"`
	Code string  `json:"Code"`
}

type AliData struct {
	OriginalPrice    float64
	DiscountPrice    float64
	Currency         string `json:"Currency,omitempty"`
	Quantity         int64
	ModuleDetails    AliModuleDetails    `json:"ModuleDetails"`
	PromotionDetails AliPromotionDetails `json:"PromotionDetails"`
	TradePrice       float64
}
type AliModuleDetails struct {
	ModuleDetail []AliModuleDetail `json:"ModuleDetail"`
}

type AliModuleDetail struct {
	UnitPrice         float64
	ModuleCode        string
	CostAfterDiscount float64
	Price             float64 `json:"OriginalCost"`
	InvoiceDiscount   float64
}

type AliPromotionDetails struct {
	PromotionDetail []AliPromotionDetail
}

type AliPromotionDetail struct {
	PromotionName string `json:"PromotionName,omitempty"`
	PromotionId   int64  `json:"PromotionId,omitempty"`
}

type AliProudctInstanceTypes struct {
	InstanceTypes AliInstanceTypes `json:"InstanceTypes"`
}
type AliInstanceTypes struct {
	InstanceType []AliInstanceType `json:"InstanceType"`
}
type AliInstanceType struct {
	InstanceType string  `json:"InstanceTypeId,omitempty"`
	Vcpu         int64   `json:"CpuCoreCount,omitempty"`
	Memory       float64 `json:"MemorySize,omitempty"`
	Gpu          string  `json:"GPUSpec,omitempty"`
	GpuAmount    int64   `json:"GpuAmount"`
	GpuSpec      string  `json:"GpuSpec"`
	GpuMemory    int64   `json:"GPUAmount,omitempty"`
}

type AliInstanceTypesResponse struct {
	RequestId     string `json:"RequestId"`
	NextToken     string `json:"NextToken"`
	InstanceTypes struct {
		InstanceType []ecs.InstanceType `json:"InstanceType"`
	} `json:"InstanceTypes"`
}

var validFilterKey map[string]bool

func init() {
	validFilterKey = make(map[string]bool, 0)

	refelectValue := reflect.ValueOf(irs.ProductInfo{})

	for i := 0; i < refelectValue.NumField(); i++ {
		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		if _, ok := validFilterKey[camelCaseFieldName]; !ok {
			validFilterKey[camelCaseFieldName] = true
		}
	}

	refelectValue = reflect.ValueOf(irs.OnDemand{})

	for i := 0; i < refelectValue.NumField(); i++ {
		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		if _, ok := validFilterKey[camelCaseFieldName]; !ok {
			validFilterKey[camelCaseFieldName] = true
		}
	}
}

func (priceInfoHandler *AlibabaPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	var familyList []string
	familyList = append(familyList, "ecs")

	return familyList, nil
}

func (priceInfoHandler *AlibabaPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	priceMap := make(map[string]irs.Price)

	priceMutex := &sync.Mutex{}

	cblogger.Debug(filterList)
	filter, _ := filterListToMap(filterList)

	cblogger.Infof("filter value : %+v", filterList)

	if filteredRegionName, ok := filter["regionName"]; ok {
		regionName = *filteredRegionName
	} else if regionName == "" {
		regionName = irs.RegionInfo{}.Region
	}

	if productFamily != "ecs" {
		return priceInfoHandler.getOtherProductFamilyPriceInfo(productFamily, regionName, filterList)
	}

	cloudPrice := irs.CloudPrice{}

	cloudPrice.Meta.Version = "0.5"
	cloudPrice.Meta.Description = "Multi-Cloud Price Info"
	cloudPrice.CloudName = "ALIBABA"
	cloudPrice.RegionName = regionName

	availableInstanceTypes := make(map[string]bool)

	availableResourceRequest := ecs.CreateDescribeAvailableResourceRequest()
	availableResourceRequest.Scheme = "https"
	availableResourceRequest.RegionId = regionName
	availableResourceRequest.DestinationResource = "InstanceType"

	availableResourceResponse, err := priceInfoHandler.EcsClient.DescribeAvailableResource(availableResourceRequest)
	if err != nil {
		cblogger.Error("Failed to get available instance types: ", err)
		return "", err
	}

	for _, availableZone := range availableResourceResponse.AvailableZones.AvailableZone {
		for _, availableResource := range availableZone.AvailableResources.AvailableResource {
			for _, supportedResource := range availableResource.SupportedResources.SupportedResource {
				availableInstanceTypes[supportedResource.Value] = true
			}
		}
	}

	if filter["instanceType"] != nil {
		if _, ok := availableInstanceTypes[*filter["instanceType"]]; !ok {
			cblogger.Warnf("Filtered instance type %s is not available in region %s", *filter["instanceType"], regionName)
			if !availableInstanceTypes[*filter["instanceType"]] {
				availableInstanceTypes = make(map[string]bool)
				availableInstanceTypes[*filter["instanceType"]] = true
			}
		} else {
			filteredType := *filter["instanceType"]
			availableInstanceTypes = make(map[string]bool)
			availableInstanceTypes[filteredType] = true
		}
	}

	cblogger.Infof("Found %d available instance types in region %s", len(availableInstanceTypes), regionName)

	if len(availableInstanceTypes) == 0 {
		cblogger.Error("No available instance types found in region: ", regionName)
		return "", fmt.Errorf("No available instance types found in region: %s", regionName)
	}

	instanceTypesRequest := ecs.CreateDescribeInstanceTypesRequest()
	instanceTypesRequest.Scheme = "https"
	instanceTypesRequest.RegionId = regionName

	if filter["vcpu"] != nil {
		vcpuInt, _ := strconv.Atoi(*filter["vcpu"])
		instanceTypesRequest.MinimumCpuCoreCount = requests.NewInteger(vcpuInt)
		instanceTypesRequest.MaximumCpuCoreCount = requests.NewInteger(vcpuInt)
	}

	if filter["memory"] != nil {
		memoryFloat, _ := strconv.ParseFloat(*filter["memory"], 64)
		instanceTypesRequest.MinimumMemorySize = requests.NewFloat(memoryFloat)
		instanceTypesRequest.MaximumMemorySize = requests.NewFloat(memoryFloat)
	}

	if filter["storage"] != nil {
		instanceTypesRequest.LocalStorageCategory = *filter["storage"]
	}

	if filter["gpu"] != nil {
		instanceTypesRequest.GPUSpec = *filter["gpu"]
	}

	if filter["gpuMemory"] != nil {
		gpuMemoryInt, _ := strconv.Atoi(*filter["gpuMemory"])
		instanceTypesRequest.MinimumGPUAmount = requests.NewInteger(gpuMemoryInt)
		instanceTypesRequest.MaximumGPUAmount = requests.NewInteger(gpuMemoryInt)
	}

	instanceTypesResponse, err := priceInfoHandler.EcsClient.DescribeInstanceTypes(instanceTypesRequest)
	if err != nil {
		cblogger.Error("Failed to get instance types: ", err)
		return "", err
	}

	instanceTypesInfo := make(map[string]ecs.InstanceType)
	for _, typeInfo := range instanceTypesResponse.InstanceTypes.InstanceType {
		instanceTypesInfo[typeInfo.InstanceTypeId] = typeInfo
	}

	filteredInstanceTypesInfo := make(map[string]ecs.InstanceType)
	for instanceType := range availableInstanceTypes {
		if typeInfo, exists := instanceTypesInfo[instanceType]; exists {
			filteredInstanceTypesInfo[instanceType] = typeInfo
		}
	}

	instanceTypesInfo = filteredInstanceTypesInfo

	cblogger.Infof("Retrieved details for %d instance types", len(instanceTypesInfo))

	convertToProductInfo := func(instanceType string, regionName string) (irs.ProductInfo, error) {
		productInfo := irs.ProductInfo{}

		instanceTypeInfo, exists := instanceTypesInfo[instanceType]
		if !exists {
			return productInfo, fmt.Errorf("instance type %s not found in cache", instanceType)
		}

		if filter["vcpu"] != nil {
			vcpuInt, _ := strconv.Atoi(*filter["vcpu"])
			if int(instanceTypeInfo.CpuCoreCount) != vcpuInt {
				return productInfo, fmt.Errorf("vcpu filter not matched")
			}
		}

		if filter["memory"] != nil {
			memoryFloat, _ := strconv.ParseFloat(*filter["memory"], 64)
			if instanceTypeInfo.MemorySize != memoryFloat {
				return productInfo, fmt.Errorf("memory filter not matched")
			}
		}

		if filter["storage"] != nil && *filter["storage"] != "" {
			if instanceTypeInfo.LocalStorageCategory != *filter["storage"] {
				return productInfo, fmt.Errorf("storage filter not matched")
			}
		}

		if filter["gpu"] != nil && *filter["gpu"] != "" {
			if instanceTypeInfo.GPUSpec != *filter["gpu"] {
				return productInfo, fmt.Errorf("gpu filter not matched")
			}
		}

		if filter["gpuMemory"] != nil {
			gpuMemoryInt, _ := strconv.Atoi(*filter["gpuMemory"])
			if int(instanceTypeInfo.GPUAmount) != gpuMemoryInt {
				return productInfo, fmt.Errorf("gpuMemory filter not matched")
			}
		}

		r := make(map[string]interface{})
		jsonBytes, _ := json.Marshal(instanceTypeInfo)
		json.Unmarshal(jsonBytes, &r)
		productInfo.CSPProductInfo = r

		productInfo.ProductId = "NA"

		simpleMode := strings.ToUpper(os.Getenv("VMSPECINFO_SIMPLE_MODE_IN_PRICEINFO")) == "ON"

		if simpleMode {
			productInfo.VMSpecName = instanceType
		} else {
			var gpuInfo []irs.GpuInfo
			if instanceTypeInfo.GPUAmount > 0 && instanceTypeInfo.GPUSpec != "" {
				gpuMemorySize := int(instanceTypeInfo.GPUMemorySize)
				totalMemory := instanceTypeInfo.GPUAmount * gpuMemorySize

				gpuInfo = []irs.GpuInfo{
					{
						Count:          strconv.Itoa(instanceTypeInfo.GPUAmount),
						Mfr:            "NA",
						Model:          instanceTypeInfo.GPUSpec,
						MemSizeGB:      strconv.Itoa(gpuMemorySize),
						TotalMemSizeGB: strconv.Itoa(totalMemory),
					},
				}
			}

			productInfo.VMSpecInfo = &irs.VMSpecInfo{
				Region:     regionName,
				Name:       instanceType,
				VCpu:       irs.VCpuInfo{Count: strconv.Itoa(instanceTypeInfo.CpuCoreCount), ClockGHz: "-1"},
				MemSizeMiB: strconv.FormatFloat(instanceTypeInfo.MemorySize*1024, 'f', 0, 64),
				DiskSizeGB: "-1",
				Gpu:        gpuInfo,
			}
		}

		return productInfo, nil
	}

	priceUnit := "Hour"

	var instanceTypeKeys []string
	for instanceType := range instanceTypesInfo {
		if filter["instanceType"] != nil && instanceType != *filter["instanceType"] {
			continue
		}
		instanceTypeKeys = append(instanceTypeKeys, instanceType)
	}

	numParts := 3
	chunkSize := (len(instanceTypeKeys) + numParts - 1) / numParts

	var wg sync.WaitGroup

	for part := 0; part < numParts; part++ {
		wg.Add(1)

		start := part * chunkSize
		end := (part + 1) * chunkSize
		if end > len(instanceTypeKeys) {
			end = len(instanceTypeKeys)
		}

		partInstanceTypes := instanceTypeKeys[start:end]

		go func(partNum int, instanceTypes []string) {
			defer wg.Done()

			cblogger.Infof("Starting goroutine %d with %d instance types", partNum, len(instanceTypes))

			delay := time.Duration(20*partNum) * time.Millisecond

			for _, instanceType := range instanceTypes {
				time.Sleep(delay)

				period := 1
				systemDiskCategories := []string{"cloud_essd", "cloud_efficiency"}

				var priceResp *ecs.DescribePriceResponse
				var priceErr error
				var usedSystemDiskCategory string

				for _, diskCategory := range systemDiskCategories {
					priceRequest := ecs.CreateDescribePriceRequest()
					priceRequest.Scheme = "https"
					priceRequest.RegionId = regionName
					priceRequest.ResourceType = "instance"
					priceRequest.InstanceType = instanceType
					priceRequest.SystemDiskCategory = diskCategory
					priceRequest.Period = requests.NewInteger(period)
					priceRequest.PriceUnit = priceUnit

					for retry := 0; retry < 3; retry++ {
						priceResp, priceErr = priceInfoHandler.EcsClient.DescribePrice(priceRequest)

						if priceErr == nil {
							break
						}

						errMsg := priceErr.Error()
						if strings.Contains(errMsg, "ErrorCode: Throttling") {
							cblogger.Warnf("Throttling error for instance type %s with disk category %s : %v",
								instanceType, diskCategory, priceErr)
							cblogger.Infof("Rate limit exceeded. Waiting 5 seconds before retrying...")
							time.Sleep(5 * time.Second)
							continue
						}
						if strings.Contains(errMsg, "ErrorCode: UnknownError") ||
							strings.Contains(errMsg, "ErrorCode: ImageNotSupportInstanceType") {
							break
						}
					}

					if priceErr == nil {
						usedSystemDiskCategory = diskCategory
						break
					}

					if diskCategory == systemDiskCategories[0] && len(systemDiskCategories) > 1 {
						cblogger.Warnf("Failed to get price with disk category %s, trying %s: %v",
							diskCategory, systemDiskCategories[1], priceErr)
					}
				}

				if priceErr != nil {
					errMsg := priceErr.Error()
					if strings.Contains(errMsg, "ErrorCode: UnknownError") ||
						strings.Contains(errMsg, "ErrorCode: ImageNotSupportInstanceType") {
						break
					}

					// InvalidSystemDiskCategory.ValueNotSupported는 정상적인 상황이므로 info 로그로 처리
					if strings.Contains(errMsg, "ErrorCode: InvalidSystemDiskCategory.ValueNotSupported") {
						cblogger.Infof("Instance type %s does not support available disk categories in region %s, skipping", instanceType, regionName)
					} else {
						cblogger.Errorf("Failed to get price for instance type %s with all disk categories: %v", instanceType, priceErr)
					}
					continue
				}

				productId := regionName + "_" + instanceType

				var instanceTypePrice float64 = 0

				if priceResp != nil && priceResp.PriceInfo.Price.DetailInfos.DetailInfo != nil {
					for _, detailInfo := range priceResp.PriceInfo.Price.DetailInfos.DetailInfo {
						if detailInfo.Resource == "instanceType" {
							instanceTypePrice = detailInfo.TradePrice
						}
					}
				}

				totalPrice := instanceTypePrice

				onDemand := irs.OnDemand{
					PricingId:   "NA",
					Unit:        priceUnit,
					Currency:    priceResp.PriceInfo.Price.Currency,
					Price:       strconv.FormatFloat(totalPrice, 'f', -1, 64),
					Description: fmt.Sprintf("Available SystemDisk: %s", usedSystemDiskCategory),
				}

				aliPriceInfo := make(map[string]interface{})
				jsonBytes, _ := json.Marshal(priceResp.PriceInfo.Price)
				json.Unmarshal(jsonBytes, &aliPriceInfo)

				priceMutex.Lock()

				aPrice, ok := priceMap[productId]
				if ok {
					aPrice.PriceInfo.OnDemand = onDemand
					aPrice.PriceInfo.CSPPriceInfo = aliPriceInfo
					priceMap[productId] = aPrice
				} else {
					newProductInfo, err := convertToProductInfo(instanceType, regionName)
					if err != nil {
						priceMutex.Unlock()
						cblogger.Errorf("[%s] instanceType error: %s", instanceType, err.Error())
						continue
					}

					newPrice := irs.Price{}
					newPrice.ZoneName = "NA"
					newProductInfo.ProductId = productId
					newProductInfo.Description = fmt.Sprintf("SystemDisk: %s", usedSystemDiskCategory)
					newPrice.ProductInfo = newProductInfo

					newPriceInfo := irs.PriceInfo{}
					newPriceInfo.OnDemand = onDemand
					newPriceInfo.CSPPriceInfo = aliPriceInfo
					newPrice.PriceInfo = newPriceInfo

					priceMap[productId] = newPrice
				}

				priceMutex.Unlock()
			}

			cblogger.Infof("Completed goroutine %d", partNum)
		}(part, partInstanceTypes)
	}

	wg.Wait()
	cblogger.Infof("All goroutines completed")

	priceList := []irs.Price{}
	for _, value := range priceMap {
		priceList = append(priceList, value)
	}

	cloudPrice.PriceList = priceList

	resultString, err := json.Marshal(cloudPrice)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return string(resultString), nil
}

func (priceInfoHandler *AlibabaPriceInfoHandler) getOtherProductFamilyPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	return "", fmt.Errorf("Non-ECS product family pricing not implemented yet")
}

func toCamelCase(val string) string {
	if val == "" {
		return ""
	}
	returnString := fmt.Sprintf("%s%s", strings.ToLower(val[:1]), val[1:])
	return returnString
}

func invalidRefelctCheck(value reflect.Value) bool {
	return value.Kind() == reflect.Array ||
		value.Kind() == reflect.Slice ||
		value.Kind() == reflect.Map ||
		value.Kind() == reflect.Func ||
		value.Kind() == reflect.Interface ||
		value.Kind() == reflect.UnsafePointer ||
		value.Kind() == reflect.Chan
}

func productInfoFilter(productInfo *irs.ProductInfo, filter map[string]*string) bool {
	if len(filter) == 0 {
		return false
	}

	refelectValue := reflect.ValueOf(*productInfo)

	for i := 0; i < refelectValue.NumField(); i++ {
		fieldName := refelectValue.Type().Field(i).Name

		if fieldName == "CSPProductInfo" || fieldName == "Description" {
			continue
		}

		camelCaseFieldName := toCamelCase(fieldName)
		fieldValue := refelectValue.Field(i)

		if invalidRefelctCheck(fieldValue) ||
			fieldValue.Kind() == reflect.Ptr ||
			fieldValue.Kind() == reflect.Struct {
			continue
		}

		fieldStringValue := fmt.Sprintf("%v", fieldValue)

		if value, ok := filter[camelCaseFieldName]; ok {
			skipFlag := value != nil && *value != fieldStringValue

			if skipFlag {
				return true
			}
		}
	}

	return false
}

func filterListToMap(filterList []irs.KeyValue) (map[string]*string, bool) {
	filterMap := make(map[string]*string, 0)

	if filterList == nil {
		return filterMap, true
	}

	for _, kv := range filterList {
		if _, ok := validFilterKey[kv.Key]; !ok {
			return map[string]*string{}, false
		}

		value := strings.TrimSpace(kv.Value)
		if value == "" {
			continue
		}

		filterMap[kv.Key] = &value
	}

	return filterMap, true
}
