// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	bssopenapi "github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi" // update to v1.62.327 from v1.61.1743, due to QuerySkuPriceListRequest struct
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaPriceInfoHandler struct {
	BssClient *bssopenapi.Client
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
	//Storage      string  `json:"LocalStorageCategory,omitempty"`
	Gpu       string `json:"GPUSpec,omitempty"`
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

	fmt.Printf("valid key is this %+v\n", validFilterKey)

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
	cblogger.Info(filterList)
	filter, _ := filterListToMap(filterList)

	cblogger.Infof("filter value : %+v", filterList)

	if filteredRegionName, ok := filter["regionName"]; ok {
		regionName = *filteredRegionName
	} else if regionName == "" {
		regionName = irs.RegionInfo{}.Region
	}

	productListresponse, err := QueryProductList(priceInfoHandler.BssClient)
	if err != nil {
		return "", err
	}

	targetProducts := []bssopenapi.Product{}
	for _, product := range productListresponse.Data.ProductList.Product {
		if product.ProductCode == productFamily {
			targetProducts = append(targetProducts, product)
		}
	} // targetProducts에는 PayAsYouGo, Subscription
	if len(targetProducts) == 0 {
		cblogger.Errorf("There is no match productFamily input - [%s]", productFamily)
		return "", err // 뉴 에러로 처리 해야 함
	}

	cloudPriceData := irs.CloudPriceData{}
	cloudPriceData.Meta.Version = "v0.1"
	cloudPriceData.Meta.Description = "Multi-Cloud Price Info"

	cloudPrice := irs.CloudPrice{}
	cloudPrice.CloudName = "Alibaba"

	for _, product := range targetProducts {

		if product.SubscriptionType == "PayAsYouGo" {
			if _, ok := filter["purchaseOption"]; ok && "PayAsYouGo" != *filter["purchaseOption"] {
				continue
			}
			pricingModuleRequest := bssopenapi.CreateDescribePricingModuleRequest()
			pricingModuleRequest.Scheme = "https"
			pricingModuleRequest.SubscriptionType = product.SubscriptionType
			pricingModuleRequest.ProductCode = product.ProductCode
			pricingModuleRequest.ProductType = product.ProductType

			pricingModulesPayAsYouGo, err := priceInfoHandler.BssClient.DescribePricingModule(pricingModuleRequest)
			if err != nil {
				cblogger.Error(err)
				continue
			}
			// cblogger.Info("pricingModuleRequest ", pricingModuleRequest)
			// cblogger.Info("pricingModulesPayAsYouGo ", pricingModulesPayAsYouGo)
			isExist := bool(false)
			var pricingModulePriceType string
			for _, pricingModule := range pricingModulesPayAsYouGo.Data.ModuleList.Module {
				// cblogger.Info("pricingModule ", pricingModule)
				if pricingModule.ModuleCode == "InstanceType" {
					for _, config := range pricingModule.ConfigList.ConfigList {
						if config == "InstanceType" {
							pricingModulePriceType = pricingModule.PriceType
							isExist = true
							break
						}
					}
					break
				}
			}

			if !isExist {
				cblogger.Errorf("There is no InstanceType Module Config - [%s]", productFamily)
				continue
				//break
			}

			getPayAsYouGoPriceRequest := requests.NewCommonRequest()
			getPayAsYouGoPriceRequest.Method = "POST"
			getPayAsYouGoPriceRequest.Scheme = "https"
			getPayAsYouGoPriceRequest.Domain = "business.ap-southeast-1.aliyuncs.com" // endPoint 고정
			getPayAsYouGoPriceRequest.Version = "2017-12-14"
			getPayAsYouGoPriceRequest.ApiName = "GetPayAsYouGoPrice"
			getPayAsYouGoPriceRequest.QueryParams["SubscriptionType"] = "PayAsYouGo"
			getPayAsYouGoPriceRequest.QueryParams["ProductCode"] = product.ProductCode
			getPayAsYouGoPriceRequest.QueryParams["ProductType"] = product.ProductType
			getPayAsYouGoPriceRequest.QueryParams["Region"] = regionName

			for _, pricingModulesAttr := range pricingModulesPayAsYouGo.Data.AttributeList.Attribute {
				if pricingModulesAttr.Code == "InstanceType" {
					for _, attr := range pricingModulesAttr.Values.AttributeValue {
						// 특정 InstanceType에 Attr 에 존재하지만, 쿼리에 InternalError 발생. 부득이 단건 호출.
						//cblogger.Infof("Now query is : [%s] %s ", product.SubscriptionType, attr.Value)

						if filter["instanceType"] != nil && attr.Value != *filter["instanceType"] {
							continue
						}

						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.ModuleCode"] = "InstanceType"
						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.PriceType"] = pricingModulePriceType // Hour
						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.Config"] = "InstanceType:" + attr.Value

						if filter["leaseContractLength"] != nil {
							getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.PriceType"] = *filter["leaseContractLength"]
						}

						priceResponse, err := priceInfoHandler.BssClient.ProcessCommonRequest(getPayAsYouGoPriceRequest)
						if err != nil {
							cblogger.Error(err.Error())
							continue
						}

						priceResp := AliPriceInfo{}
						priceResponseStr := priceResponse.GetHttpContentString()
						err = json.Unmarshal([]byte(priceResponseStr), &priceResp)
						if err != nil {
							cblogger.Error(err.Error())
							continue
						}

						if filteredCurrency, ok := filter["currency"]; ok {
							if priceResp.Data.Currency != *filteredCurrency {
								cblogger.Info(priceResp.Data.Currency + ":" + *filteredCurrency)
								continue
							}
						}

						if filteredPrice, ok := filter["price"]; ok {
							priceNum, _ := strconv.ParseFloat(*filteredPrice, 0)
							if priceResp.Data.ModuleDetails.ModuleDetail[0].Price != priceNum {
								//cblogger.Info(priceResp.Data.ModuleDetail[0].Price , *filteredCurrency)
								continue
							}
						}

						pricingPolicy, err := BindpricingPolicy(priceResp, product.SubscriptionType, pricingModulePriceType, regionName, attr.Value)
						if err != nil {
							cblogger.Error(err.Error())
							continue
						}

						productId := regionName + "_" + attr.Value
						// product : price = 1 : 1
						// price : price policy = 1 : n
						aPrice, ok := priceMap[productId]
						if ok { // product가 존재하면 policy 추가
							aPrice.PriceInfo.PricingPolicies = append(aPrice.PriceInfo.PricingPolicies, pricingPolicy)
							aPrice.PriceInfo.CSPPriceInfo = append(aPrice.PriceInfo.CSPPriceInfo.([]string), priceResponseStr)
							priceMap[productId] = aPrice
						} else { // product가 없으면 price 추가
							newProductInfo, err := GetDescribeInstanceTypesForPricing(priceInfoHandler.BssClient, regionName, attr.Value, filter)
							if err != nil {
								cblogger.Errorf("[%s] instanceType is Empty", attr.Value)
								continue
							}

							newPriceInfo := irs.PriceInfo{}
							newPolicies := []irs.PricingPolicies{}
							newPolicies = append(newPolicies, pricingPolicy)

							newPriceInfo.PricingPolicies = newPolicies
							newCSPPriceInfo := []string{}
							newCSPPriceInfo = append(newCSPPriceInfo, priceResponseStr)
							newPriceInfo.CSPPriceInfo = newCSPPriceInfo

							newPrice := irs.Price{}
							newPrice.PriceInfo = newPriceInfo
							newPrice.ProductInfo = newProductInfo

							priceMap[productId] = newPrice
						}
					}
				}
			}
		} else if product.SubscriptionType == "Subscription" {
			if _, ok := filter["purchaseOption"]; ok && "Subscription" != *filter["purchaseOption"] {
				continue
			}
			pricingModuleRequest := bssopenapi.CreateDescribePricingModuleRequest()
			pricingModuleRequest.Scheme = "https"
			pricingModuleRequest.SubscriptionType = product.SubscriptionType
			pricingModuleRequest.ProductCode = product.ProductCode
			pricingModuleRequest.ProductType = product.ProductType

			pricingModulesSubscription, err := priceInfoHandler.BssClient.DescribePricingModule(pricingModuleRequest)
			if err != nil {
				cblogger.Error(err)
				return "", err
			}

			isExist := bool(false)
			for _, pricingModule := range pricingModulesSubscription.Data.ModuleList.Module {
				if pricingModule.ModuleCode == "InstanceType" {
					for _, config := range pricingModule.ConfigList.ConfigList {
						if config == "InstanceType" {
							isExist = true
						}
					}
				}
			}
			if !isExist {
				cblogger.Errorf("There is no InstanceType Module Config - [%s]", productFamily)
				continue
			}
			pricingModulePriceTypes := []string{"Month", "Year"}
			if filteredpricingModulePriceTypes, ok := filter["leaseContractLength"]; ok { //filter key = leaseContractLength, 결과 : unit
				pricingModulePriceTypes = []string{*filteredpricingModulePriceTypes}
			}
			for _, pricingModulePriceType := range pricingModulePriceTypes {
				getSubscriptionPrice := requests.NewCommonRequest()
				getSubscriptionPrice.Method = "POST"
				getSubscriptionPrice.Scheme = "https" // https | http
				getSubscriptionPrice.Domain = "business.ap-southeast-1.aliyuncs.com"
				getSubscriptionPrice.Version = "2017-12-14"
				getSubscriptionPrice.ApiName = "GetSubscriptionPrice"

				getSubscriptionPrice.QueryParams["SubscriptionType"] = product.SubscriptionType //"Subscription"
				getSubscriptionPrice.QueryParams["ProductCode"] = product.ProductCode           //"ecs"
				getSubscriptionPrice.QueryParams["OrderType"] = "NewOrder"                      // NewOrder, Upgrade, Renewal
				getSubscriptionPrice.QueryParams["ServicePeriodUnit"] = pricingModulePriceType
				getSubscriptionPrice.QueryParams["ServicePeriodQuantity"] = "1" // 1 초과시 PromotionDetails 응답 없음

				for _, pricingModulesAttr := range pricingModulesSubscription.Data.AttributeList.Attribute {

					if pricingModulesAttr.Code == "InstanceType" {
						for _, attr := range pricingModulesAttr.Values.AttributeValue {
							//cblogger.Infof("Now query is : [%s] %s ", product.SubscriptionType, attr.Value)

							// 특정 InstanceType에 Attr 에 존재하지만, 쿼리에 InternalError 발생. 부득이 단건 호출.
							getSubscriptionPrice.QueryParams["ModuleList.1.ModuleCode"] = "InstanceType"
							getSubscriptionPrice.QueryParams["ModuleList.1.Config"] = "InstanceType:" + attr.Value

							if filter["instanceType"] != nil && attr.Value != *filter["instanceType"] {
								continue
							}

							priceResponse, err := priceInfoHandler.BssClient.ProcessCommonRequest(getSubscriptionPrice)
							if err != nil {
								cblogger.Error(err.Error())
								continue
							}
							priceResp := AliPriceInfo{}
							priceResponseStr := priceResponse.GetHttpContentString()
							err = json.Unmarshal([]byte(priceResponseStr), &priceResp)
							if err != nil {
								cblogger.Error(err.Error())
								continue
							}

							pricingPolicy, err := BindpricingPolicy(priceResp, product.SubscriptionType, pricingModulePriceType, regionName, attr.Value)
							if err != nil {
								cblogger.Error(err.Error())
								continue
							}
							productId := regionName + "_" + attr.Value

							aPrice, ok := priceMap[productId]
							if ok { // product가 존재하면 policy 추가
								aPrice.PriceInfo.PricingPolicies = append(aPrice.PriceInfo.PricingPolicies, pricingPolicy)
								aPrice.PriceInfo.CSPPriceInfo = append(aPrice.PriceInfo.CSPPriceInfo.([]string), priceResponseStr)
								priceMap[productId] = aPrice
							} else { // product가 없으면 price 추가
								newProductInfo, err := GetDescribeInstanceTypesForPricing(priceInfoHandler.BssClient, regionName, attr.Value, filter)
								if err != nil {
									cblogger.Errorf("[%s] instanceType is Empty", attr.Value)
									continue
								}

								newPriceInfo := irs.PriceInfo{}
								newPolicies := []irs.PricingPolicies{}
								newPolicies = append(newPolicies, pricingPolicy)

								newPriceInfo.PricingPolicies = newPolicies
								newCSPPriceInfo := []string{}
								newCSPPriceInfo = append(newCSPPriceInfo, priceResponseStr)
								newPriceInfo.CSPPriceInfo = newCSPPriceInfo

								newPrice := irs.Price{}
								newPrice.PriceInfo = newPriceInfo
								newPrice.ProductInfo = newProductInfo

								priceMap[productId] = newPrice
							}

						}
					}
				}
			}
		}
	}

	// priceMap 을 List 로 반환
	priceList := []irs.Price{}
	for _, value := range priceMap {
		priceList = append(priceList, value)
	}

	cloudPrice.PriceList = priceList
	cloudPriceData.CloudPriceList = append(cloudPriceData.CloudPriceList, cloudPrice)

	resultString, err := json.Marshal(cloudPriceData)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return string(resultString), nil
}

// Util start

// region의 특정 instanceType의 내용조회
// func GetDescribeInstanceTypesForPricing(bssClient *bssopenapi.Client, instanceType string) (map[string]interface{}, error) {
func GetDescribeInstanceTypesForPricing(bssClient *bssopenapi.Client, regionName string, instanceType string, filter map[string]*string) (irs.ProductInfo, error) {
	DescribeInstanceRequest := requests.NewCommonRequest()
	DescribeInstanceRequest.Method = "POST"
	DescribeInstanceRequest.Scheme = "https" // https | http
	DescribeInstanceRequest.Domain = "ecs.ap-southeast-1.aliyuncs.com"
	DescribeInstanceRequest.Version = "2014-05-26"
	DescribeInstanceRequest.ApiName = "DescribeInstanceTypes"
	DescribeInstanceRequest.QueryParams["InstanceTypes.1"] = instanceType

	if filter["instanceType"] != nil {
		DescribeInstanceRequest.QueryParams["InstanceTypes.1"] = *filter["instanceType"]
	}

	if filter["vcpu"] != nil {
		DescribeInstanceRequest.QueryParams["MinimumCpuCoreCount"] = *filter["vcpu"]
		DescribeInstanceRequest.QueryParams["MaximumCpuCoreCount"] = *filter["vcpu"]
	}

	if filter["memory"] != nil {
		DescribeInstanceRequest.QueryParams["MinimumMemorySize"] = *filter["memory"]
		DescribeInstanceRequest.QueryParams["MaximumMemorySize"] = *filter["memory"]
	}

	if filter["storage"] != nil {
		DescribeInstanceRequest.QueryParams["LocalStorageCategory"] = *filter["storage"]
	}

	if filter["gpu"] != nil {
		DescribeInstanceRequest.QueryParams["GPUSpec"] = *filter["gpu"]
	}

	if filter["gpuMemory"] != nil {
		DescribeInstanceRequest.QueryParams["MinimumGPUAmount"] = *filter["gpuMemory"]
		DescribeInstanceRequest.QueryParams["MaximumGPUAmount"] = *filter["gpuMemory"]
	}

	productInfo := irs.ProductInfo{}
	instanceResponse, err := bssClient.ProcessCommonRequest(DescribeInstanceRequest)
	if err != nil {
		cblogger.Error(err.Error())
		return productInfo, err
	}
	instanceResp := AliProudctInstanceTypes{}
	err = json.Unmarshal([]byte(instanceResponse.GetHttpContentString()), &instanceResp)
	if err != nil {
		cblogger.Errorf("Error parsing JSON:%s", err.Error())
	} else {
		if len(instanceResp.InstanceTypes.InstanceType) > 0 {
			resultProduct := instanceResp.InstanceTypes.InstanceType[0]
			productInfo.CSPProductInfo = instanceResponse.GetHttpContentString()
			productInfo.ProductId = "NA" //regionName + "_" + instanceType
			productInfo.RegionName = regionName
			productInfo.ZoneName = "NA"
			productInfo.InstanceType = resultProduct.InstanceType
			productInfo.Vcpu = strconv.FormatInt(resultProduct.Vcpu, 10)
			productInfo.Memory = strconv.FormatFloat(resultProduct.Memory, 'f', -1, 64)
			//resultProduct.Storage 데이터가 없는데 왜 데이터를 이렇게 맵핑을 해놓았는지 확인필요
			//productInfo.Storage = resultProduct.Storage
			productInfo.Storage = "NA"
			productInfo.Gpu = resultProduct.Gpu
			productInfo.GpuMemory = strconv.FormatInt(resultProduct.GpuMemory, 10)
			productInfo.OperatingSystem = "NA"
			productInfo.PreInstalledSw = "NA"
			productInfo.VolumeType = "NA"
			productInfo.StorageMedia = "NA"
			productInfo.MaxVolumeSize = "NA"
			productInfo.MaxIOPSVolume = "NA"
			productInfo.MaxThroughputVolume = "NA"
			productInfo.Description = "NA"
		} else {
			return productInfo, errors.New("there is no instanceType")
		}
	}

	return productInfo, nil
}

func BindpricingPolicy(priceResp AliPriceInfo, subscriptionType string, pricingModulePriceType string, regionName string, instanceType string) (irs.PricingPolicies, error) {

	pricingPolicy := irs.PricingPolicies{}
	pricingPolicy.PricingId = regionName + "_" + instanceType + "_" + subscriptionType + "_" + pricingModulePriceType //"NA"
	pricingPolicy.PricingPolicy = subscriptionType
	pricingPolicy.Unit = "NA"
	pricingPolicy.Currency = priceResp.Data.Currency
	if len(priceResp.Data.ModuleDetails.ModuleDetail) > 0 {
		resultModuleDetailPrice := priceResp.Data.ModuleDetails.ModuleDetail[0]
		pricingPolicy.Price = strconv.FormatFloat(resultModuleDetailPrice.Price, 'f', -1, 64)
	} else {
		return irs.PricingPolicies{}, errors.New("No Price Data")
	}

	if len(priceResp.Data.PromotionDetails.PromotionDetail) > 0 {
		pricingPolicy.Description = fmt.Sprintf("%s(%d)", priceResp.Data.PromotionDetails.PromotionDetail[0].PromotionName, priceResp.Data.PromotionDetails.PromotionDetail[0].PromotionId)
	} else {
		pricingPolicy.Description = "NA"

	}
	pricingPolicy.PricingPolicyInfo = &irs.PricingPolicyInfo{
		LeaseContractLength: pricingModulePriceType,
		OfferingClass:       "NA",
		PurchaseOption:      "NA",
	}

	return pricingPolicy, nil
}

func toCamelCase(val string) string {
	if val == "" {
		return ""
	}

	return fmt.Sprintf("%s%s", strings.ToLower(val[:1]), val[1:])
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
