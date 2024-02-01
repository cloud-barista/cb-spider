package resources

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/pricing"
)

type AwsPriceInfoHandler struct {
	Region idrv.RegionInfo
	Client *pricing.Pricing
}

// AWS에서는 Region이 Product list에 영향을 주지 않습니다.
// 3개 Region Endpoint에서만 Product 정보를 리턴합니다.
// getPricingClient에 Client *pricing.Pricing 정의
func (priceInfoHandler *AwsPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	var result []string
	input := &pricing.GetAttributeValuesInput{
		AttributeName: aws.String("productfamily"),
		MaxResults:    aws.Int64(32), // 2024.01 기준 32개
		ServiceCode:   aws.String("AmazonEC2"),
	}

	cblogger.Info("input 1321312434242341312312", input)
	for {
		attributeValues, err := priceInfoHandler.Client.GetAttributeValues(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case pricing.ErrCodeInternalErrorException:
					cblogger.Error(pricing.ErrCodeInternalErrorException, aerr.Error())
				case pricing.ErrCodeInvalidParameterException:
					cblogger.Error(pricing.ErrCodeInvalidParameterException, aerr.Error())
				case pricing.ErrCodeNotFoundException:
					cblogger.Error(pricing.ErrCodeNotFoundException, aerr.Error())
				case pricing.ErrCodeInvalidNextTokenException:
					cblogger.Error(pricing.ErrCodeInvalidNextTokenException, aerr.Error())
				case pricing.ErrCodeExpiredNextTokenException:
					cblogger.Error(pricing.ErrCodeExpiredNextTokenException, aerr.Error())
				default:
					cblogger.Error(aerr.Error())
				}
			} else {
				// Prnit the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				cblogger.Error(err.Error())
			}
		}

		for _, attributeValue := range attributeValues.AttributeValues {

			//result = append(result, *attributeValue.Value)
			result = append(result, *attributeValue.Value)
		}

		for i := range attributeValues.AttributeValues {
			result[i] = removeSpaces(result[i])
		}

		for _, attributeValue := range attributeValues.AttributeValues {
			attributeValue.Value = aws.String(strings.ReplaceAll(*attributeValue.Value, " ", ""))
		}

		// 결과 출력
		cblogger.Info("rkskekfkekfkekfkekf", attributeValues)
		fmt.Printf("%+v\n", attributeValues)

		cblogger.Info("attributeValue0000000000000000000000000000", attributeValues.AttributeValues)

		cblogger.Info("attributeValue===============================", result)
		if attributeValues.NextToken != nil {
			input = &pricing.GetAttributeValuesInput{
				NextToken: attributeValues.NextToken,
			}
		} else {
			break
		}
	}

	return result, nil
}
func removeSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

// AWS에서는 ListProductFamily를 통해 ProductFamily와 AttributeName을 수집하고,
// GetAttributeValues를 통해 AttributeValue를 수집하여 필터로 사용합니다.
// GetPriceInfo는 DescribeServices를 통해 올바른 productFamily 인자만 검사합니다. -> AttributeName에 오류가 있을경우 빈값을 리턴

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
func (priceInfoHandler *AwsPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, additionalFilterList []irs.KeyValue) (string, error) {
	filter, _ := filterListToMap(additionalFilterList)
	cblogger.Infof("productFamily======", productFamily)

	cblogger.Infof("filter value : %+v", additionalFilterList)

	describeServicesinput := &pricing.DescribeServicesInput{
		ServiceCode: aws.String(productFamily),
		MaxResults:  aws.Int64(1),
	}

	cblogger.Info("describeServicesinput", describeServicesinput)

	services, err := priceInfoHandler.Client.DescribeServices(describeServicesinput)
	if services == nil {
		cblogger.Error("No services in given productFamily. CHECK productFamily!")
		return "", err
	}
	// for the test
	cblogger.Info("services55555555555", services)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	//getProductsinputfilters := []*pricing.Filter{}

	// if filterList != nil {
	// 	for _, filter := range filterList {
	// 		var getProductsinputfilter pricing.Filter

	// 		err := json.Unmarshal([]byte(filter.Value), &getProductsinputfilter)
	// 		getProductsinputfilters = append(getProductsinputfilters, &getProductsinputfilter)

	// 		if err != nil {
	// 			cblogger.Error(err)
	// 			return "", err
	// 		}
	// 		// for the test
	// 		cblogger.Info("getProductsinputfilter", getProductsinputfilter)
	// 	}

	// 	// for the test
	// 	cblogger.Info("[]*pricing.Filter{}", []*pricing.Filter{})
	// }
	// if regionName != "" {
	// 	getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
	// 		Field: aws.String("regionCode"),
	// 		Type:  aws.String("EQUALS"),
	// 		Value: aws.String(regionName),
	// 	})
	// }

	// getProductsinput := &pricing.GetProductsInput{
	// 	Filters:     getProductsinputfilters,
	// 	ServiceCode: aws.String(productFamily),
	// }

	// additionalFilterList []*pricing.Filter로 변환
	var filters []*pricing.Filter

	for _, kv := range additionalFilterList {
		filter := &pricing.Filter{
			Field: aws.String(kv.Key),
			Value: aws.String(kv.Value),
		}
		filters = append(filters, filter)
	}

	getProductsinput := &pricing.GetProductsInput{
		Filters:     filters,
		ServiceCode: aws.String(productFamily),
	}

	priceinfos, err := priceInfoHandler.Client.GetProducts(getProductsinput)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	result := &irs.CloudPriceData{}
	result.Meta.Version = "v0.1"
	result.Meta.Description = "Multi-Cloud Price Info"

	for _, price := range priceinfos.PriceList {
		jsonString, err := json.MarshalIndent(price["product"].(map[string]interface{})["attributes"], "", "    ")
		if err != nil {
			cblogger.Error(err)
		}

		var productInfo irs.ProductInfo
		ReplaceEmptyWithNA(&productInfo)
		err = json.Unmarshal(jsonString, &productInfo)
		if err != nil {
			cblogger.Error(err)
		}

		productInfo.ProductId = fmt.Sprintf("%s", price["product"].(map[string]interface{})["sku"])
		productInfo.RegionName = fmt.Sprintf("%s", price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["regionCode"])
		productInfo.Description = fmt.Sprintf("productFamily= %s, version= %s", price["product"].(map[string]interface{})["productFamily"], price["version"])
		productInfo.CSPProductInfo = price["product"]
		productInfo.ZoneName = "NA" // AWS zone is Not Applicable - 202401

		var priceInfo irs.PriceInfo
		priceInfo.CSPPriceInfo = price["terms"]
		for termsKey, termsValue := range price["terms"].(map[string]interface{}) {
			for _, policyvalue := range termsValue.(map[string]interface{}) {
				var pricingPolicy irs.PricingPolicies
				for innerpolicyKey, innerpolicyValue := range policyvalue.(map[string]interface{}) {
					if innerpolicyKey == "priceDimensions" {
						for priceDimensionsKey, priceDimensionsValue := range innerpolicyValue.(map[string]interface{}) {
							pricingPolicy.PricingId = priceDimensionsKey
							pricingPolicy.PricingPolicy = termsKey
							pricingPolicy.Description = fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["description"])
							for key, val := range priceDimensionsValue.(map[string]interface{})["pricePerUnit"].(map[string]interface{}) {
								pricingPolicy.Currency = key
								pricingPolicy.Price = fmt.Sprintf("%s", val)
								// USD is Default.
								// if NO USD data, accept other currency.
								if key == "USD" {
									break
								}
							}
							if filter["productId"] != nil && productInfo.ProductId != *filter["productId"] {
								continue
							}
							if filter["regionName"] != nil && productInfo.RegionName != *filter["regionName"] {
								continue
							}
							if filter["instanceType"] != nil && productInfo.InstanceType != *filter["instanceType"] {
								continue
							}
							if filter["vcpu"] != nil && productInfo.Vcpu != *filter["vcpu"] {
								continue
							}
							if filter["memory"] != nil && productInfo.Memory != *filter["memory"] {
								continue
							}
							if filter["storage"] != nil && productInfo.Storage != *filter["storage"] {
								continue
							}
							if filter["gpu"] != nil && productInfo.Gpu != *filter["gpu"] {
								continue
							}
							if filter["gpuMemory"] != nil && productInfo.GpuMemory != *filter["gpuMemory"] {
								continue
							}
							if filter["operatingSystem"] != nil && productInfo.OperatingSystem != *filter["operatingSystem"] {
								continue
							}
							if filter["preInstalledSw"] != nil && productInfo.PreInstalledSw != *filter["preInstalledSw"] {
								continue
							}

							pricingPolicy.Unit = fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["unit"])

							if filter["unit"] != nil && pricingPolicy.Unit != *filter["unit"] {
								continue
							}
							if filter["pricingId"] != nil && pricingPolicy.PricingId != *filter["pricingId"] {
								continue
							}
							if filter["pricingPolicy"] != nil && pricingPolicy.PricingPolicy != *filter["pricingPolicy"] {
								continue
							}
							if filter["currency"] != nil && pricingPolicy.Currency != *filter["currency"] {
								continue
							}
							if filter["price"] != nil && pricingPolicy.Price != *filter["price"] {
								continue
							}
							if filter["description"] != nil && pricingPolicy.Description != *filter["description"] {
								continue
							}
							if filter["leaseContractLength"] != nil && pricingPolicy.PricingPolicyInfo.LeaseContractLength != *filter["leaseContractLength"] {
								continue
							}
							if filter["offeringClass"] != nil && pricingPolicy.PricingPolicyInfo.OfferingClass != *filter["offeringClass"] {
								continue
							}
							if filter["purchaseOption"] != nil && pricingPolicy.PricingPolicyInfo.PurchaseOption != *filter["purchaseOption"] {
								continue
							}

							priceInfo.PricingPolicies = append(priceInfo.PricingPolicies, pricingPolicy)
						}
					}
				}
			}
		}

		// price info
		var priceListone irs.Price
		priceListone.ProductInfo = productInfo
		priceListone.PriceInfo = priceInfo

		priceone := irs.CloudPrice{
			CloudName: "AWS",
		}
		priceone.PriceList = append(priceone.PriceList, priceListone)
		result.CloudPriceList = append(result.CloudPriceList, priceone)
	}

	resultString, err := json.Marshal(result)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return string(resultString), nil
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
func filterListToMap(additionalFilterList []irs.KeyValue) (map[string]*string, bool) { // 키-값 목록을 받아서 필터링 된 맵과 유효성 검사 결과 반환
	filterMap := make(map[string]*string, 0) // 빈 맵 생성

	if additionalFilterList == nil { // 입력값이 nil이면 빈 맵과 true를 반환합니다.
		return filterMap, true
	}

	for _, kv := range additionalFilterList { // 각 키-값 쌍에 대해 다음 작업 수행
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
