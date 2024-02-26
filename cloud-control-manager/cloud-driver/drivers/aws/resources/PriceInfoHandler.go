package resources

import (
	"encoding/json"
	"fmt"

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
			result = append(result, *attributeValue.Value)
		}
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

// AWS에서는 ListProductFamily를 통해 ProductFamily와 AttributeName을 수집하고,
// GetAttributeValues를 통해 AttributeValue를 수집하여 필터로 사용합니다.
// GetPriceInfo는 DescribeServices를 통해 올바른 productFamily 인자만 검사합니다. -> AttributeName에 오류가 있을경우 빈값을 리턴

func (priceInfoHandler *AwsPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	result := &irs.CloudPriceData{}
	result.Meta.Version = "v0.1"
	result.Meta.Description = "Multi-Cloud Price Info"

	priceMap := make(map[string]irs.Price) // 전체 price를 id로 구분한 map

	cblogger.Info("productFamily : ", productFamily)
	cblogger.Info("filter value : ", filterList)
	requestProductsInputFilters, err := setProductsInputRequestFilter(filterList)

	// filter조건에 region 지정.
	if regionName != "" {
		requestProductsInputFilters = append(requestProductsInputFilters, &pricing.Filter{
			Field: aws.String("regionCode"),
			Type:  aws.String("EQUALS"),
			Value: aws.String(regionName),
		})
	} else {
		requestProductsInputFilters = append(requestProductsInputFilters, &pricing.Filter{
			Field: aws.String("regionCode"),
			Type:  aws.String("EQUALS"),
			Value: aws.String(priceInfoHandler.Region.Region),
		})
	}

	requestProductsInputFilters = append(requestProductsInputFilters, &pricing.Filter{
		Field: aws.String("productFamily"),
		Type:  aws.String("EQUALS"),
		Value: aws.String(productFamily),
	})
	cblogger.Info("requestProductsInputFilters", requestProductsInputFilters)

	// NextToken 설정 x -> 1 ~ 100개 까지 출력
	// NextToken값이 있으면 request에 NextToken값을 추가
	// NextToken값이 없어질 때 까지 반복
	// Region : us-west-1 / ProductFamily : Compute Instance 조회 결과 39200개 확인
	var nextToken *string
	for {

		getProductsRequest := &pricing.GetProductsInput{
			Filters:     requestProductsInputFilters,
			ServiceCode: aws.String("AmazonEC2"), // ServiceCode : AmazonEC2 고정
			NextToken:   nextToken,
		}
		cblogger.Info("get Products request", getProductsRequest)

		priceInfos, err := priceInfoHandler.Client.GetProducts(getProductsRequest)
		if err != nil {
			cblogger.Error(err)
			return "", err
		}
		cblogger.Info("get Products response", priceInfos)

		for _, awsPrice := range priceInfos.PriceList {
			productInfo, err := ExtractProductInfo(awsPrice, productFamily)

			if err != nil {
				cblogger.Error(err)
				continue
			}

			// termsKey : OnDemand, Reserved
			for termsKey, termsValue := range awsPrice["terms"].(map[string]interface{}) {
				for _, policyValue := range termsValue.(map[string]interface{}) {
					// OnDemand, Reserved 일 때, 항목이 다름.
					priceDemensions := make(map[string]interface{})
					termAttributes := make(map[string]interface{})
					sku := ""
					if priceDemensionsVal, ok := policyValue.(map[string]interface{})["priceDimensions"]; ok {
						priceDemensions = priceDemensionsVal.(map[string]interface{})
					}
					if termAttributesVal, ok := policyValue.(map[string]interface{})["termAttributes"]; ok {
						termAttributes = termAttributesVal.(map[string]interface{})
					}
					if skuVal, ok := policyValue.(map[string]interface{})["sku"]; ok {
						skuValString, ok := skuVal.(string)
						if ok {
							sku = skuValString
						}
					}

					if termsKey == "OnDemand" {
						for priceDimensionsKey, priceDimensionsValue := range priceDemensions {
							isFiltered := OnDemandPolicyFilter(priceDimensionsKey, priceDimensionsValue.(map[string]interface{}), termAttributes, sku, filterList)
							if isFiltered {
								continue
							}

							var pricingPolicy irs.PricingPolicies
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
							pricingPolicy.Unit = fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["unit"])

							// policy 추출하여 추가
							aPrice, ok := AppendPolicyToPrice(priceMap, productInfo, pricingPolicy, awsPrice)
							if !ok {
								priceMap[productInfo.ProductId] = aPrice
							}
						}

					} else if termsKey == "Reserved" {

						for priceDimensionsKey, priceDimensionsValue := range priceDemensions {
							isFiltered := ReservedPolicyFilter(priceDimensionsKey, priceDimensionsValue.(map[string]interface{}), termAttributes, sku, filterList)
							if isFiltered {
								continue
							}

							var pricingPolicy irs.PricingPolicies
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
							pricingPolicy.Unit = fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["unit"])

							pricingPolicyInfo := irs.PricingPolicyInfo{}
							if leaseContractLength, ok := termAttributes["LeaseContractLength"]; ok {
								pricingPolicyInfo.LeaseContractLength = leaseContractLength.(string)
							}
							if offeringClass, ok := termAttributes["OfferingClass"]; ok {
								pricingPolicyInfo.OfferingClass = offeringClass.(string)
							}
							if purchaseOption, ok := termAttributes["PurchaseOption"]; ok {
								pricingPolicyInfo.PurchaseOption = purchaseOption.(string)
							}

							// policy 추출하여 추가
							aPrice, ok := AppendPolicyToPrice(priceMap, productInfo, pricingPolicy, awsPrice)
							if !ok {
								priceMap[productInfo.ProductId] = aPrice
							}
						}
					}
				}
			}
		}

		// nextToken이 없으면 for Loop 중단.
		if priceInfos.NextToken == nil {
			break
		}
		// NextToken값이 있다면 설정
		nextToken = priceInfos.NextToken
	} // end of nextToken for

	priceList := []irs.Price{}
	for _, value := range priceMap {
		priceList = append(priceList, value)
	}
	priceone := irs.CloudPrice{
		CloudName: "AWS",
	}

	priceone.PriceList = priceList
	result.CloudPriceList = append(result.CloudPriceList, priceone)
	resultString, err := json.Marshal(result)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}
	return string(resultString), nil
}

// 가져온 결과에서 product 추출
func ExtractProductInfo(jsonValue aws.JSONValue, productFamily string) (irs.ProductInfo, error) {
	var productInfo irs.ProductInfo

	jsonString, err := json.MarshalIndent(jsonValue["product"].(map[string]interface{})["attributes"], "", "    ")
	if err != nil {
		cblogger.Error(err)
		return productInfo, err
	}
	switch productFamily {
	case "Compute Instance":
		ReplaceEmptyWithNAforComputeInstance(&productInfo)
	case "Storage":
		ReplaceEmptyWithNAforStorage(&productInfo)
	case "Load Balancer-Network":
		ReplaceEmptyWithNAforLoadBalancerNetwork(&productInfo)
	default:
		ReplaceEmptyWithNA(&productInfo)
	}

	err = json.Unmarshal(jsonString, &productInfo)
	if err != nil {
		cblogger.Error(err)
		return productInfo, err
	}

	productId := fmt.Sprintf("%s", jsonValue["product"].(map[string]interface{})["sku"])
	productInfo.ProductId = productId
	productInfo.RegionName = fmt.Sprintf("%s", jsonValue["product"].(map[string]interface{})["attributes"].(map[string]interface{})["regionCode"])
	productInfo.Description = fmt.Sprintf("productFamily= %s, version= %s", jsonValue["product"].(map[string]interface{})["productFamily"], jsonValue["version"])
	productInfo.CSPProductInfo = jsonValue["product"]
	productInfo.ZoneName = "NA" // AWS zone is Not Applicable - 202401

	return productInfo, nil

}

// price에 해당 product가 있으면 append, 없으면 추가
func AppendPolicyToPrice(priceMap map[string]irs.Price, productInfo irs.ProductInfo, pricingPolicy irs.PricingPolicies, jsonValue aws.JSONValue) (irs.Price, bool) {
	productId := productInfo.ProductId
	aPrice, ok := priceMap[productId]

	if ok { // product가 존재하면 policy 추가
		cblogger.Info("product exist ", productId)
		aPrice.PriceInfo.PricingPolicies = append(aPrice.PriceInfo.PricingPolicies, pricingPolicy)
		priceMap[productId] = aPrice
		return aPrice, true
	} else { // product가 없으면 price 추가
		cblogger.Info("product not exist ", productId)

		newPriceInfo := irs.PriceInfo{}
		newPolicies := []irs.PricingPolicies{}
		newPolicies = append(newPolicies, pricingPolicy)

		newPriceInfo.PricingPolicies = newPolicies
		newPriceInfo.CSPPriceInfo = jsonValue // 새로운 가격이면 terms아래값을 넣는다.

		newPrice := irs.Price{}
		newPrice.PriceInfo = newPriceInfo
		newPrice.ProductInfo = productInfo

		priceMap[productId] = newPrice

		return newPrice, true
	}
}

// 요청시 필요한 filter Set.
func setProductsInputRequestFilter(filterList []irs.KeyValue) ([]*pricing.Filter, error) {
	requestFilters := []*pricing.Filter{}

	if filterList != nil {
		for _, filter := range filterList {
			if filter.Key == "instanceType" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("instanceType"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}

			if filter.Key == "operatingSystem" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("operatingSystem"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "vcpu" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("vcpu"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "productId" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("sku"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "memory" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("memory"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "storage" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("storage"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "gpu" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("gpu"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "gpuMemory" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("gpuMemory"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "preInstalledSw" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Field: aws.String("preInstalledSw"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
		} //end of for
	} // end of if
	return requestFilters, nil
}

// 결과에서 filter. filter에 걸리면 true, 안걸리면 false
func OnDemandPolicyFilter(priceDimensionsKey string, priceDimensions map[string]interface{}, termAttributes map[string]interface{}, sku string, filterList []irs.KeyValue) bool {
	isFiltered := false

	hasPricingPolicy := false
	pricingPolicyVal := ""

	hasPriceDimension := false
	priceDemensionVal := ""

	hasUnit := false
	unitVal := ""

	// reserved only options
	hasLeaseContractLength := false
	hasOfferingClass := false
	hasPurchaseOption := false

	if filterList != nil {

		for _, filter := range filterList {
			// find filter conditions
			if filter.Key == "pricingPolicy" {
				hasPricingPolicy = true
				pricingPolicyVal = filter.Value
				continue
			}

			if filter.Key == "pricingId" {
				hasPriceDimension = true
				priceDemensionVal = filter.Value
				continue
			}
			if filter.Key == "unit" {
				hasUnit = true
				unitVal = filter.Value
				continue
			}
			if filter.Key == "leaseContractLength" {
				hasLeaseContractLength = true
				break
			}
			if filter.Key == "offeringClass" {
				hasOfferingClass = true
				break
			}
			if filter.Key == "purchaseOption" {
				hasPurchaseOption = true
				break
			}
		}
		// check filters
	}

	if hasLeaseContractLength || hasOfferingClass || hasPurchaseOption { // reserved 전용 filter 임.
		cblogger.Info("filtered by reserved options ", hasLeaseContractLength, hasOfferingClass, hasPurchaseOption)
		return true
	}

	if hasPricingPolicy {
		if pricingPolicyVal != "OnDemand" {
			cblogger.Info("filtered by pricingPolicy ", pricingPolicyVal, priceDimensions["pricingPolicy"])
			return true
		}
	}
	if hasUnit {
		for key, val := range priceDimensions["pricePerUnit"].(map[string]interface{}) {
			// USD is Default.
			// if NO USD data, accept other currency.
			if key == "USD" {
				if unitVal != val {
					cblogger.Info("filtered by price per unit ", unitVal, priceDimensions["pricePerUnit"])
					return true
				}
				break
			}
		}
	}

	if hasPriceDimension {
		if priceDemensionVal != priceDimensionsKey { // priceId
			cblogger.Info("filtered by priceDimension ", priceDemensionVal, priceDimensionsKey)
			return true
		}
	}
	return isFiltered
}

// filter에 걸리면 true
func ReservedPolicyFilter(priceDimensionsKey string, priceDimensionsValue map[string]interface{}, termAttributes map[string]interface{}, sku string, filterList []irs.KeyValue) bool {
	isFiltered := false

	hasPricingPolicy := false
	pricingPolicyVal := ""

	hasPriceDimension := false
	priceDemensionVal := ""

	hasUnit := false
	unitVal := ""

	hasLeaseContractLength := false
	leaseContractLengthVal := ""

	hasOfferingClass := false
	offeringClassVal := ""

	hasPurchaseOption := false
	purchaseOptionVal := ""

	if filterList != nil {

		for _, filter := range filterList {
			// find filter conditions
			if filter.Key == "pricingPolicy" {
				hasPricingPolicy = true
				pricingPolicyVal = filter.Value
				continue
			}

			if filter.Key == "pricingId" {
				hasPriceDimension = true
				priceDemensionVal = filter.Value
				continue
			}
			if filter.Key == "unit" {
				hasUnit = true
				unitVal = filter.Value
				continue
			}
			if filter.Key == "leaseContractLength" {
				hasLeaseContractLength = true
				leaseContractLengthVal = filter.Value
				continue
			}
			if filter.Key == "offeringClass" {
				hasOfferingClass = true
				offeringClassVal = filter.Value
				continue
			}
			if filter.Key == "purchaseOption" {
				hasPurchaseOption = true
				purchaseOptionVal = filter.Value
				continue
			}
		}
	}

	if hasPriceDimension {
		if priceDemensionVal != priceDimensionsKey { // priceId
			return true
		}
	}

	if hasPricingPolicy {
		if pricingPolicyVal != "Reserved" {
			return true
		}
	}
	if hasUnit {
		for key, val := range priceDimensionsValue["pricePerUnit"].(map[string]interface{}) {
			// USD is Default.
			// if NO USD data, accept other currency.
			if key == "USD" {
				if unitVal != val {
					return true
				}
				break
			}

		}
	}

	if hasLeaseContractLength {
		if leaseContractLengthVal != termAttributes["LeaseContractLength"] {
			return true
		}
	}

	if hasOfferingClass {
		if offeringClassVal != termAttributes["OfferingClass"] {
			return true
		}
	}
	if hasPurchaseOption {
		if purchaseOptionVal != termAttributes["PurchaseOption"] {
			return true
		}
	}
	return isFiltered
}
