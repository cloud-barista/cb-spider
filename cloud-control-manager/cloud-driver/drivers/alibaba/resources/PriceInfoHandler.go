// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	bssopenapi "github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi" // update to v1.62.327 from v1.61.1743, due to QuerySkuPriceListRequest struct
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaPriceInfoHandler struct {
	BssClient *bssopenapi.Client
	EcsClient *ecs.Client
}

// Alibaba  GetSubscriptionPrice struct start

// Alibaba GetSubscriptionPrice Response Data 최상위
type AliPriceInfo struct {
	Data AliData `json:"Data"`
	Code string  `json:"Code"`
}

// Alibaba GetSubscriptionPrice ModuleDetails, PromotionDetails 배열 데이터 존재
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

// Alibaba ServicePeriodUnit par 값을 Year로 넘길시 프로모션 관련 정보도 리턴을 해줌.
type AliPromotionDetails struct {
	PromotionDetail []AliPromotionDetail
}

type AliPromotionDetail struct {
	PromotionName string `json:"PromotionName,omitempty"`
	PromotionId   int64  `json:"PromotionId,omitempty"`
}

// Alibaba GetSubscriptionPrice end

// Alibaba DescribeInstanceTypes Response start
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
	// Storage      string  `json:"LocalStorageCategory,omitempty"` 잘못된 데이터 맵핑
	Gpu       string `json:"GPUSpec,omitempty"`
	GpuAmount int64  `json:"GpuAmount"`
	GpuSpec   string `json:"GpuSpec"`
	GpuMemory int64  `json:"GPUAmount,omitempty"`
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

	refelectValue := reflect.ValueOf(irs.ProductInfo{}) // 구조체의 reflect 값을 가져옴

	for i := 0; i < refelectValue.NumField(); i++ {

		fieldName := refelectValue.Type().Field(i).Name       // 구조체의 필드 이름을 가져옴
		camelCaseFieldName := toCamelCase(fieldName)          // 필드 이름을 CamelCase로 변환합니다.
		if _, ok := validFilterKey[camelCaseFieldName]; !ok { // 맵에 이미해당 필드 이름이 존재하지 않으면 맵에 추가
			validFilterKey[camelCaseFieldName] = true
		}
	}

	refelectValue = reflect.ValueOf(irs.PricingPolicies{}) // ris.PricingPolicies 구조체에 동일한 작업 반복

	for i := 0; i < refelectValue.NumField(); i++ {

		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		if _, ok := validFilterKey[camelCaseFieldName]; !ok {
			validFilterKey[camelCaseFieldName] = true
		}
	}

	refelectValue = reflect.ValueOf(irs.PricingPolicyInfo{})

	for i := 0; i < refelectValue.NumField(); i++ {

		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		if _, ok := validFilterKey[camelCaseFieldName]; !ok {
			validFilterKey[camelCaseFieldName] = true
		}
	}

	//fmt.Printf("valid key is this %+v\n", validFilterKey)

}
func (priceInfoHandler *AlibabaPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	var familyList []string
	familyList = append(familyList, "ecs") //spider에서 지원하는 가격 서비스

	// productListresponse, err := QueryProductList(priceInfoHandler.BssClient)
	// if err != nil {
	// 	return nil, err
	// }
	// for _, Product := range productListresponse.Data.ProductList.Product {
	// 	familyList = append(familyList, Product.ProductCode)
	// }

	return familyList, nil
}

func (priceInfoHandler *AlibabaPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	priceMap := make(map[string]irs.Price)

	// mutex for thread safety
	priceMutex := &sync.Mutex{}

	cblogger.Debug(filterList)
	filter, _ := filterListToMap(filterList)

	cblogger.Infof("filter value : %+v", filterList)

	// Set region from filter or use default
	if filteredRegionName, ok := filter["regionName"]; ok {
		regionName = *filteredRegionName
	} else if regionName == "" {
		regionName = irs.RegionInfo{}.Region
	}

	// Handle non-ECS product families separately
	if productFamily != "ecs" {
		return priceInfoHandler.getOtherProductFamilyPriceInfo(productFamily, regionName, filterList)
	}

	// Initialize price data structures
	cloudPriceData := irs.CloudPriceData{}
	cloudPriceData.Meta.Version = "v0.1"
	cloudPriceData.Meta.Description = "Multi-Cloud Price Info"

	cloudPrice := irs.CloudPrice{}
	cloudPrice.CloudName = "Alibaba"

	// Get available instance types
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

	// Extract available instance types
	for _, availableZone := range availableResourceResponse.AvailableZones.AvailableZone {
		for _, availableResource := range availableZone.AvailableResources.AvailableResource {
			for _, supportedResource := range availableResource.SupportedResources.SupportedResource {
				availableInstanceTypes[supportedResource.Value] = true
			}
		}
	}

	// Apply instance type filter if specified
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

	// Return error if no instance types available
	if len(availableInstanceTypes) == 0 {
		cblogger.Error("No available instance types found in region: ", regionName)
		return "", fmt.Errorf("No available instance types found in region: %s", regionName)
	}

	// Get instance type details
	instanceTypesRequest := ecs.CreateDescribeInstanceTypesRequest()
	instanceTypesRequest.Scheme = "https"
	instanceTypesRequest.RegionId = regionName

	// Apply filters to request
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

	// Map instance types for easier access
	instanceTypesInfo := make(map[string]ecs.InstanceType)
	for _, typeInfo := range instanceTypesResponse.InstanceTypes.InstanceType {
		instanceTypesInfo[typeInfo.InstanceTypeId] = typeInfo
	}

	// Filter by available instance types
	filteredInstanceTypesInfo := make(map[string]ecs.InstanceType)
	for instanceType := range availableInstanceTypes {
		if typeInfo, exists := instanceTypesInfo[instanceType]; exists {
			filteredInstanceTypesInfo[instanceType] = typeInfo
		}
	}

	instanceTypesInfo = filteredInstanceTypesInfo

	cblogger.Infof("Retrieved details for %d instance types", len(instanceTypesInfo))

	// Function to convert instance type to product info
	convertToProductInfo := func(instanceType string, region string) (irs.ProductInfo, error) {
		productInfo := irs.ProductInfo{}

		instanceTypeInfo, exists := instanceTypesInfo[instanceType]
		if !exists {
			return productInfo, fmt.Errorf("instance type %s not found in cache", instanceType)
		}

		// Apply additional filters
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

		// Build product info
		r := make(map[string]interface{})
		jsonBytes, _ := json.Marshal(instanceTypeInfo)
		json.Unmarshal(jsonBytes, &r)
		productInfo.CSPProductInfo = r

		productInfo.ProductId = "NA"
		productInfo.RegionName = region
		productInfo.ZoneName = "NA"
		productInfo.VMSpecInfo.Name = instanceTypeInfo.InstanceTypeId
		productInfo.VMSpecInfo.VCpu.Count = strconv.Itoa(int(instanceTypeInfo.CpuCoreCount))
		productInfo.VMSpecInfo.VCpu.ClockGHz = "-1"
		productInfo.VMSpecInfo.MemSizeMiB = strconv.FormatFloat(instanceTypeInfo.MemorySize*1024, 'f', -1, 64)
		productInfo.VMSpecInfo.DiskSizeGB = "-1"

		// Add GPU info if applicable
		if instanceTypeInfo.GPUAmount > 0 {
			productInfo.VMSpecInfo.Gpu = []irs.GpuInfo{
				{
					Count:          strconv.Itoa(int(instanceTypeInfo.GPUAmount)),
					MemSizeGB:      strconv.FormatInt(int64(instanceTypeInfo.GPUMemorySize), 10),
					TotalMemSizeGB: strconv.FormatInt(int64(instanceTypeInfo.GPUMemorySize*float64(instanceTypeInfo.GPUAmount)), 10),
					Mfr:            "NA",
					Model:          instanceTypeInfo.GPUSpec,
				},
			}
		}

		productInfo.PreInstalledSw = "NA"

		if productFamily != "ecs" {
			productInfo.VolumeType = "NA"
			productInfo.StorageMedia = "NA"
			productInfo.MaxVolumeSize = "NA"
			productInfo.MaxIOPSVolume = "NA"
			productInfo.MaxThroughputVolume = "NA"
		}
		productInfo.Description = "NA"

		return productInfo, nil
	}

	// Set price unit to Hour only
	priceUnit := "Hour"

	// Convert instance types to slice for parallel processing
	var instanceTypeKeys []string
	for instanceType := range instanceTypesInfo {
		if filter["instanceType"] != nil && instanceType != *filter["instanceType"] {
			continue
		}
		instanceTypeKeys = append(instanceTypeKeys, instanceType)
	}

	// Split work for parallel processing
	numParts := 3
	chunkSize := (len(instanceTypeKeys) + numParts - 1) / numParts

	var wg sync.WaitGroup

	// Launch goroutines to fetch prices in parallel
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

				// Try different disk categories
				for _, diskCategory := range systemDiskCategories {
					priceRequest := ecs.CreateDescribePriceRequest()
					priceRequest.Scheme = "https"
					priceRequest.RegionId = regionName
					priceRequest.ResourceType = "instance"
					priceRequest.InstanceType = instanceType
					priceRequest.SystemDiskCategory = diskCategory
					priceRequest.Period = requests.NewInteger(period)
					priceRequest.PriceUnit = priceUnit

					// Retry logic for API calls
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

					// Store successful disk category
					if priceErr == nil {
						usedSystemDiskCategory = diskCategory
						break
					}

					// Log failure of first disk category
					if diskCategory == systemDiskCategories[0] && len(systemDiskCategories) > 1 {
						cblogger.Warnf("Failed to get price with disk category %s, trying %s: %v",
							diskCategory, systemDiskCategories[1], priceErr)
					}
				}

				// Skip if all disk categories failed
				if priceErr != nil {
					errMsg := priceErr.Error()
					if strings.Contains(errMsg, "ErrorCode: UnknownError") ||
						strings.Contains(errMsg, "ErrorCode: ImageNotSupportInstanceType") {
						break
					}
					cblogger.Errorf("Failed to get price for instance type %s with all disk categories: %v", instanceType, priceErr)
					continue
				}

				// Create product ID based on region and instance type
				productId := regionName + "_" + instanceType

				// Set purchase option to OnDemand for Hour pricing
				purchaseOption := "OnDemand"

				// Extract instance type price from response
				var instanceTypePrice float64 = 0

				if priceResp != nil && priceResp.PriceInfo.Price.DetailInfos.DetailInfo != nil {
					for _, detailInfo := range priceResp.PriceInfo.Price.DetailInfos.DetailInfo {
						if detailInfo.Resource == "instanceType" {
							instanceTypePrice = detailInfo.TradePrice
						}
					}
				}

				// Use instance price
				totalPrice := instanceTypePrice

				// Create pricing policy
				pricingPolicy := irs.PricingPolicies{
					PricingId:     "NA",
					PricingPolicy: purchaseOption,
					Unit:          priceUnit,
					Currency:      priceResp.PriceInfo.Price.Currency,
					Price:         strconv.FormatFloat(totalPrice, 'f', -1, 64),
					Description:   fmt.Sprintf("Available SystemDisk: %s", usedSystemDiskCategory),
				}

				// Store CSP-specific price info
				aliPriceInfo := make(map[string]interface{})
				jsonBytes, _ := json.Marshal(priceResp.PriceInfo.Price)
				json.Unmarshal(jsonBytes, &aliPriceInfo)

				// Thread-safe price map update
				priceMutex.Lock()

				// Add to existing product or create new one
				aPrice, ok := priceMap[productId]
				if ok { // Add pricing policy to existing product
					aPrice.PriceInfo.PricingPolicies = append(aPrice.PriceInfo.PricingPolicies, pricingPolicy)
					aPrice.PriceInfo.CSPPriceInfo = aliPriceInfo
					priceMap[productId] = aPrice
				} else { // Create new price entry
					newProductInfo, err := convertToProductInfo(instanceType, regionName)
					if err != nil {
						priceMutex.Unlock()
						cblogger.Errorf("[%s] instanceType error: %s", instanceType, err.Error())
						continue
					}

					// Create new price entry
					newPrice := irs.Price{}
					newProductInfo.ProductId = productId
					newProductInfo.Description = fmt.Sprintf("SystemDisk: %s", usedSystemDiskCategory)
					newPrice.ProductInfo = newProductInfo

					// Set price info
					newPriceInfo := irs.PriceInfo{}
					newPolicies := []irs.PricingPolicies{pricingPolicy}
					newPriceInfo.PricingPolicies = newPolicies
					newPriceInfo.CSPPriceInfo = aliPriceInfo
					newPrice.PriceInfo = newPriceInfo

					priceMap[productId] = newPrice
				}

				priceMutex.Unlock()
			}

			cblogger.Infof("Completed goroutine %d", partNum)
		}(part, partInstanceTypes)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	cblogger.Infof("All goroutines completed")

	// Convert map to list
	priceList := []irs.Price{}
	for _, value := range priceMap {
		priceList = append(priceList, value)
	}

	// Build final response
	cloudPrice.PriceList = priceList
	cloudPriceData.CloudPriceList = append(cloudPriceData.CloudPriceList, cloudPrice)

	resultString, err := json.Marshal(cloudPriceData)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return string(resultString), nil
}

// exception handling for other product family except ecs
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

func pricePolicyInfoFilter(policy interface{}, filter map[string]*string) bool {
	if len(filter) == 0 {
		return false
	}

	refelectValue := reflect.ValueOf(policy)

	for i := 0; i < refelectValue.NumField(); i++ {

		fieldName := refelectValue.Type().Field(i).Name
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

func priceInfoFilter(policy irs.PricingPolicies, filter map[string]*string) bool {
	if len(filter) == 0 {
		return false
	}

	refelectValue := reflect.ValueOf(policy)

	for i := 0; i < refelectValue.NumField(); i++ {
		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		fieldValue := refelectValue.Field(i)

		if invalidRefelctCheck(fieldValue) ||
			fieldValue.Kind() == reflect.Struct {
			continue
		} else if fieldValue.Kind() == reflect.Ptr {

			derefernceValue := fieldValue.Elem()

			if derefernceValue.Kind() == reflect.Invalid {
				skipFlag := pricePolicyInfoFilter(irs.PricingPolicyInfo{}, filter)
				if skipFlag {
					return true
				}
			} else if derefernceValue.Kind() == reflect.Struct {
				if derefernceValue.Type().Name() == "PricingPolicyInfo" {
					skipFlag := pricePolicyInfoFilter(*policy.PricingPolicyInfo, filter)
					if skipFlag {
						return true
					}
				}
			}
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

func filterListToMap(filterList []irs.KeyValue) (map[string]*string, bool) { // 키-값 목록을 받아서 필터링 된 맵과 유효성 검사 결과 반환
	filterMap := make(map[string]*string, 0) // 빈 맵 생성

	if filterList == nil { // 입력값이 nil이면 빈 맵과 true를 반환합니다.
		return filterMap, true
	}

	for _, kv := range filterList { // 각 키-값 쌍에 대해 다음 작업 수행
		if _, ok := validFilterKey[kv.Key]; !ok { // 키가 유효한 필터 키 목록에 존재하지 않으면 빈 맵과 false를 반환합니다.
			return map[string]*string{}, false
		}

		value := strings.TrimSpace(kv.Value) // 값의 앞뒤 공백을 제거합니다.
		if value == "" {                     // 값이 빈 문자열이면 다음 키-값 쌍으로 넘어갑니다.
			continue
		}

		filterMap[kv.Key] = &value // 맵에 키와 값을 저장합니다.
	}

	return filterMap, true // 필터링된 맵과 true를 반환합니다.
}

// 유효한 필터 키 목록을 기반으로 키-값 쌍 목록을 필터링합니다
// 빈 값은 필터링에서 제외
// 필터링 된 결과를 맵 형태로 반환하고 유효성 검사 결과도 함께 반환

// Util end
