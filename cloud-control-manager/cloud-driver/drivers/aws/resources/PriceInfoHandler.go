package resources

import (
	"encoding/json"
	"fmt"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/aws/aws-sdk-go/aws"

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
	result = append(result, "AmazonEC2")
	// input := &pricing.GetAttributeValuesInput{
	// 	AttributeName: aws.String("productfamily"),
	// 	MaxResults:    aws.Int64(32), // 2024.01 기준 32개
	// 	ServiceCode:   aws.String("AmazonEC2"),
	// }

	// cblogger.Info("input 1321312434242341312312", input)
	// for {
	// 	attributeValues, err := priceInfoHandler.Client.GetAttributeValues(input)
	// 	if err != nil {
	// 		if aerr, ok := err.(awserr.Error); ok {
	// 			switch aerr.Code() {
	// 			case pricing.ErrCodeInternalErrorException:
	// 				cblogger.Error(pricing.ErrCodeInternalErrorException, aerr.Error())
	// 			case pricing.ErrCodeInvalidParameterException:
	// 				cblogger.Error(pricing.ErrCodeInvalidParameterException, aerr.Error())
	// 			case pricing.ErrCodeNotFoundException:
	// 				cblogger.Error(pricing.ErrCodeNotFoundException, aerr.Error())
	// 			case pricing.ErrCodeInvalidNextTokenException:
	// 				cblogger.Error(pricing.ErrCodeInvalidNextTokenException, aerr.Error())
	// 			case pricing.ErrCodeExpiredNextTokenException:
	// 				cblogger.Error(pricing.ErrCodeExpiredNextTokenException, aerr.Error())
	// 			default:
	// 				cblogger.Error(aerr.Error())
	// 			}
	// 		} else {
	// 			// Prnit the error, cast err to awserr.Error to get the Code and
	// 			// Message from an error.
	// 			cblogger.Error(err.Error())
	// 		}
	// 	}

	// 	for _, attributeValue := range attributeValues.AttributeValues {

	// 		//result = append(result, *attributeValue.Value)
	// 		result = append(result, *attributeValue.Value)
	// 	}

	// 	for i := range attributeValues.AttributeValues {
	// 		result[i] = removeSpaces(result[i])
	// 	}

	// 	for _, attributeValue := range attributeValues.AttributeValues {
	// 		attributeValue.Value = aws.String(strings.ReplaceAll(*attributeValue.Value, " ", ""))
	// 	}

	// 	// 결과 출력
	// 	cblogger.Info("rkskekfkekfkekfkekf", attributeValues)
	// 	fmt.Printf("%+v\n", attributeValues)

	// 	cblogger.Info("attributeValue0000000000000000000000000000", attributeValues.AttributeValues)

	// 	cblogger.Info("attributeValue===============================", result)
	// 	if attributeValues.NextToken != nil {
	// 		input = &pricing.GetAttributeValuesInput{
	// 			NextToken: attributeValues.NextToken,
	// 		}
	// 	} else {
	// 		break
	// 	}
	// }

	return result, nil
}
func removeSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

// AWS에서는 ListProductFamily를 통해 ProductFamily와 AttributeName을 수집하고,
// GetAttributeValues를 통해 AttributeValue를 수집하여 필터로 사용합니다.
// GetPriceInfo는 DescribeServices를 통해 올바른 productFamily 인자만 검사합니다. -> AttributeName에 오류가 있을경우 빈값을 리턴

func (priceInfoHandler *AwsPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	result := &irs.CloudPriceData{}
	result.Meta.Version = "v0.1"
	result.Meta.Description = "Multi-Cloud Price Info"

	priceMap := make(map[string]irs.Price) // 전체 price를 id로 구분한 map

	cblogger.Infof("productFamily======", productFamily)

	cblogger.Infof("filter value : %+v", filterList)
	describeServicesinput := &pricing.DescribeServicesInput{
		ServiceCode: aws.String(productFamily),
		MaxResults:  aws.Int64(1),
	}
	// for the test
	// cblogger.Info("describeServicesinput", describeServicesinput)

	services, err := priceInfoHandler.Client.DescribeServices(describeServicesinput)
	if services == nil {
		cblogger.Error("No services in given productFamily. CHECK productFamily!")
		return "", err
	}

	if err != nil {
		cblogger.Error(err)
		return "", err
	}

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

	getProductsRequest := &pricing.GetProductsInput{
		Filters:     requestProductsInputFilters,
		ServiceCode: aws.String(productFamily),
	}
	cblogger.Info("get Products request", getProductsRequest)
	priceinfos, err := priceInfoHandler.Client.GetProducts(getProductsRequest)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}
	cblogger.Info("get Products response", priceinfos)

	// for the test
	// cblogger.Info("productInfo", priceinfos)
	for _, awsPrice := range priceinfos.PriceList {
		productInfo, err := ExtractProductInfo(awsPrice)
		if err != nil {
			cblogger.Error(err)
			continue
		}

		// termsKey : OnDemand, Reserved
		for termsKey, termsValue := range awsPrice["terms"].(map[string]interface{}) {
			cblogger.Info("now termsKey = ", termsKey)
			// hasPricingPolicyVal := false
			// pricingPolicyVal := ""

			// hasPriceDimension := false
			// priceDemensionVal := ""

			// hasunit := false
			// unitVal := ""

			// hasLeaseContractLength := false
			// leaseContractLengthVal := ""

			// hasOfferingClass := false
			// offeringClassVal := ""

			// hasPurchaseOption := false
			// purchaseOptionVal := ""

			// if filterList != nil {

			// 	for _, filter := range filterList {
			// 		// find filter conditions
			// 		if filter.Key == "pricingPolicy" {
			// 			hasPricingPolicyVal = true
			// 			pricingPolicyVal = filter.Value
			// 			continue
			// 		}

			// 		if filter.Key == "pricingId" {
			// 			hasPriceDimension = true
			// 			priceDemensionVal = filter.Value
			// 			continue
			// 		}
			// 		if filter.Key == "unit" {
			// 			hasunit = true
			// 			unitVal = filter.Value
			// 			continue
			// 		}
			// 		if filter.Key == "leaseContractLength" {
			// 			hasLeaseContractLength = true
			// 			leaseContractLengthVal = filter.Value
			// 			continue
			// 		}
			// 		if filter.Key == "offeringClass" {
			// 			hasOfferingClass = true
			// 			offeringClassVal = filter.Value
			// 			continue
			// 		}
			// 		if filter.Key == "purchaseOption" {
			// 			hasPurchaseOption = true
			// 			purchaseOptionVal = filter.Value
			// 			continue
			// 		}
			// 	}
			// 	// check filters
			// 	if hasPricingPolicyVal && pricingPolicyVal != termsKey {
			// 		cblogger.Info("filtered by pricingPolicy ", pricingPolicyVal, termsKey)
			// 		continue
			// 	}
			// }

			// 아래 filter조건이 있을 때 ondemand 면 바로 skip
			//cblogger.Info(termsKey, hasLeaseContractLength, hasOfferingClass, hasPurchaseOption)
			// if termsKey == "OnDemand" {
			// 	if hasLeaseContractLength || hasOfferingClass || hasPurchaseOption {
			// 		cblogger.Info("filtered by Reserved filters ", hasLeaseContractLength, hasOfferingClass, hasPurchaseOption)
			// 		continue
			// 	}
			// }

			for _, policyValue := range termsValue.(map[string]interface{}) {
				//cblogger.Info("termsValue(((((((", termsValue) // OnDemand 밑 map

				// map이므로 항목을 추출하여 사용하자
				// OnDemand, Reserved 일 때, 항목이 다름.
				priceDemensions := make(map[string]interface{})
				termAttributes := make(map[string]interface{})
				sku := ""
				if priceDemensionsVal, ok := policyValue.(map[string]interface{})["priceDimensions"]; ok {
					cblogger.Info("priceDimensions ", priceDemensionsVal)
					priceDemensions = priceDemensionsVal.(map[string]interface{})
				}
				if termAttributesVal, ok := policyValue.(map[string]interface{})["termAttributes"]; ok {
					cblogger.Info("termAttributes ", termAttributesVal)
					termAttributes = termAttributesVal.(map[string]interface{})
				}
				if skuVal, ok := policyValue.(map[string]interface{})["sku"]; ok {
					cblogger.Info("skuVal ", skuVal)
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

						// cblogger.Info("termAttributes>>> ", termAttributes)
						// cblogger.Info("LeaseContractLength>>> ", termAttributes["LeaseContractLength"])

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

				// var pricingPolicy irs.PricingPolicies
				// for innerpolicyKey, innerpolicyValue := range policyValue.(map[string]interface{}) {
				// 	//cblogger.Info("policyvalue %%%%%%%%%%%%", policyvalue.(map[string]interface{})) // here
				// 	cblogger.Info("innerpolicyKey ?????????? ", innerpolicyKey)     // termAttribute
				// 	cblogger.Info("innerpolicyValue !!!!!!!!!! ", innerpolicyValue) // map

				// 	// 제품 1개에 대한 비용 부과 단위가 여러개 가능. ex) Hrs, Quantity
				// 	// 제품에 대한 terAttribute와는 무관, 즉 제품:Dimensions:termAttributes = 1: n : 1
				// 	if innerpolicyKey == "priceDimensions" {
				// 		for priceDimensionsKey, priceDimensionsValue := range innerpolicyValue.(map[string]interface{}) {
				// 			cblogger.Info("priceDimensionsValue))))))))))))", priceDimensionsValue)

				// 			pricingPolicy.PricingId = priceDimensionsKey
				// 			pricingPolicy.PricingPolicy = termsKey
				// 			pricingPolicy.Description = fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["description"])
				// 			for key, val := range priceDimensionsValue.(map[string]interface{})["pricePerUnit"].(map[string]interface{}) {
				// 				pricingPolicy.Currency = key
				// 				pricingPolicy.Price = fmt.Sprintf("%s", val)
				// 				// USD is Default.
				// 				// if NO USD data, accept other currency.
				// 				if key == "USD" {
				// 					break
				// 				}

				// 			}
				// 			pricingPolicy.Unit = fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["unit"])

				// 		} // end of for
				// 	} // end of priceDimensions

				// 	if innerpolicyKey == "termAttributes" {
				// 		cblogger.Info("terms ::: ", termsKey)
				// 		cblogger.Info("termAttribute ::: ", innerpolicyValue)
				// 		// innerpolicyMap := innerpolicyValue.(map[string]interface{})
				// 		// if value, ok := innerpolicyMap["LeaseContractLength"]; ok {
				// 		// 	leaseContractLength, ok := value.(string)
				// 		// 	// 변수에 값이 성공적으로 담겼을 경우
				// 		// 	if ok {
				// 		// 		if hasLeaseContractLength && leaseContractLengthVal != value {
				// 		// 			continue
				// 		// 		}
				// 		// 		cblogger.Info("LeaseContractLength 변수:", leaseContractLength)
				// 		// 		//pricingPolicy.PricingPolicyInfo.LeaseContractLength = leaseContractLength
				// 		// 	} else { // 타입 단언에 실패한 경우
				// 		// 		cblogger.Info("LeaseContractLength의 데이터 타입을 가져올 수 없습니다.")
				// 		// 		continue
				// 		// 	}
				// 		// } else {
				// 		// 	// "LeaseContractLength" 키가 존재하지 않는 경우
				// 		// 	cblogger.Info("LeaseContractLength 키가 존재하지 않습니다. ", innerpolicyValue)
				// 		// 	continue
				// 		// }

				// 		// cblogger.Info("offeringClassVal = offeringClassVal ", offeringClassVal)
				// 		// cblogger.Info("purchaseOptionVal = offeringClassVal ", purchaseOptionVal)
				// 		// map[LeaseContractLength:1yr OfferingClass:convertible PurchaseOption:No Upfront]

				// 		// pricingPolicy.PricingPolicyInfo.LeaseContractLength = fmt.Sprintf("%s", termAttributesValue.(map[string]interface{})["LeaseContractLength"])
				// 		// pricingPolicy.PricingPolicyInfo.OfferingClass = fmt.Sprint("%s", innerpolicyValue.(map[string]interface{})["OfferingClass"])
				// 		// pricingPolicy.PricingPolicyInfo.PurchaseOption = fmt.Sprint("%s", innerpolicyValue.(map[string]interface{})["PurchaseOption"])

				// 		// for termAttributeskey, termAttributesValue := range innerpolicyValue.(map[string]interface{}) {
				// 		// 	cblogger.Info("termAttributeskey!********", termAttributeskey)
				// 		// 	cblogger.Info("termAttributesValue2////////", termAttributesValue) // TODO : map이 아니라 string값임.

				// 		// if hasLeaseContractLength && leaseContractLengthVal !=
				// 		// foundSku := false
				// 		// cblogger.Info("go sku ", termAttributesValue)
				// 		// for _, skukey := range termAttributesValue.(map[string]interface{}) {
				// 		// 	cblogger.Info("skukey ", skukey)
				// 		// 	// check filters
				// 		// 	if hasLeaseContractLength && LeaseContractLengthVal == skukey {
				// 		// 		foundSku = true
				// 		// 		break
				// 		// 	}
				// 		// 	if hasOfferingClass && OfferingClassVal == skukey {
				// 		// 		foundSku = true
				// 		// 		break
				// 		// 	}
				// 		// 	if hasPurchaseOption && PurchaseOptionVal == skukey {
				// 		// 		foundSku = true
				// 		// 		break
				// 		// 	}
				// 		// 	cblogger.Info("end skukey ", skukey)

				// 		// }
				// 		// if hasLeaseContractLength && !foundSku { // sku를 못 찾았으면 skip.
				// 		// 	cblogger.Info("filtered by Sku ", hasLeaseContractLength, foundSku)
				// 		// 	continue
				// 		// }
				// 		// if hasOfferingClass && !foundSku { // sku를 못 찾았으면 skip.
				// 		// 	cblogger.Info("filtered by Sku ", hasOfferingClass, foundSku)
				// 		// 	continue
				// 		// }
				// 		// if hasPurchaseOption && !foundSku { // sku를 못 찾았으면 skip.
				// 		// 	cblogger.Info("filtered by Sku ", hasPurchaseOption, foundSku)
				// 		// 	continue
				// 		// }

				// 		// cblogger.Info("set leaseContractLength ", termAttributesValue.(map[string]interface{})["LeaseContractLength"])
				// 		// 		spew.Dump("termAttributesValue.(map[string]interface{})[LeaseContractLength]5555555555", termAttributesValue.(map[string]interface{})["LeaseContractLength"])
				// 		// 		spew.Dump("termAttributesValue6666666666", termAttributesValue)
				// 		// pricingPolicy.PricingPolicyInfo.LeaseContractLength = fmt.Sprintf("%s", termAttributesValue.(map[string]interface{})["LeaseContractLength"])
				// 		// pricingPolicy.PricingPolicyInfo.OfferingClass = fmt.Sprint("%s", innerpolicyValue.(map[string]interface{})["OfferingClass"])
				// 		// pricingPolicy.PricingPolicyInfo.PurchaseOption = fmt.Sprint("%s", innerpolicyValue.(map[string]interface{})["PurchaseOption"])

				// 		// }

				// 		//cblogger.Info("LeaseContractLength !@!@!@!@!@", pricingPolicy.PricingPolicyInfo.LeaseContractLength)
				// 	}

				// } // end of innerpolicyKey

				// aPrice, ok := priceMap[productId]

				// if ok { // product가 존재하면 policy 추가
				// 	cblogger.Info("product exist ", productId)
				// 	aPrice.PriceInfo.PricingPolicies = append(aPrice.PriceInfo.PricingPolicies, pricingPolicy)
				// 	// aPrice.PriceInfo.CSPPriceInfo = append(aPrice.PriceInfo.CSPPriceInfo.([]string), cspPriceInfo...)
				// 	// var priceInfo irs.PriceInfo
				// 	// priceInfo.CSPPriceInfo = price["terms"]
				// 	priceMap[productId] = aPrice

				// } else { // product가 없으면 price 추가
				// 	cblogger.Info("product not exist ", productId)

				// 	newPriceInfo := irs.PriceInfo{}
				// 	newPolicies := []irs.PricingPolicies{}
				// 	newPolicies = append(newPolicies, pricingPolicy)

				// 	newPriceInfo.PricingPolicies = newPolicies
				// 	newPriceInfo.CSPPriceInfo = price["terms"] // 새로운 가격이면 terms아래값을 넣는다.

				// 	// newCSPPriceInfo := []string{}
				// 	// newCSPPriceInfo = append(newCSPPriceInfo, priceResponseStr)
				// 	// newPriceInfo.CSPPriceInfo = newCSPPriceIn
				// 	newPrice := irs.Price{}
				// 	newPrice.PriceInfo = newPriceInfo
				// 	newPrice.ProductInfo = productInfo

				// 	priceMap[productId] = newPrice
				// }
			}
		}

		// price info
		// var priceListone irs.Price
		// priceListone.ProductInfo = productInfo
		// priceListone.PriceInfo = priceInfo
	}

	priceList := []irs.Price{}
	for _, value := range priceMap {
		priceList = append(priceList, value)
	}

	priceone := irs.CloudPrice{
		CloudName: "AWS",
	}
	// priceone.PriceList = append(priceone.PriceList, priceList...)
	priceone.PriceList = priceList
	result.CloudPriceList = append(result.CloudPriceList, priceone)

	resultString, err := json.Marshal(result)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}
	cblogger.Info("return ", string(resultString))
	return string(resultString), nil
}

// 가져온 결과에서 product 추출
func ExtractProductInfo(jsonValue aws.JSONValue) (irs.ProductInfo, error) {
	var productInfo irs.ProductInfo

	cblogger.Info("=-=-=-=-=-=-=-=-", jsonValue)
	jsonString, err := json.MarshalIndent(jsonValue["product"].(map[string]interface{})["attributes"], "", "    ")
	if err != nil {
		cblogger.Error(err)
		return productInfo, err
	}

	ReplaceEmptyWithNA(&productInfo)
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
			// if filter.Key == "leaseContractLength" {
			// 	requestFilters = append(requestFilters, &pricing.Filter{
			// 		Field: aws.String("leaseContractLength"),
			// 		Type:  aws.String("TERM_MATCH"),
			// 		Value: aws.String(filter.Value),
			// 	})
			// }

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
	//leaseContractLengthVal := ""
	hasOfferingClass := false
	//offeringClassVal := ""
	hasPurchaseOption := false
	//purchaseOptionVal := ""

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
				// leaseContractLengthVal = filter.Value
				break
			}
			if filter.Key == "offeringClass" {
				hasOfferingClass = true
				// offeringClassVal = filter.Value
				break
			}
			if filter.Key == "purchaseOption" {
				hasPurchaseOption = true
				// purchaseOptionVal = filter.Value
				break
			}
		}
		// check filters

	}

	if hasLeaseContractLength || hasOfferingClass || hasPurchaseOption { // reserved 전용 filter 임.
		cblogger.Info("filtered by reserved options ", hasLeaseContractLength, hasOfferingClass, hasPurchaseOption)
		return true
	}

	if hasPricingPolicy { //
		//if pricingPolicyVal != priceDimensions["pricingPolicy"] {
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

	cblogger.Info("1.pricingPolicyVal ", pricingPolicyVal, hasPricingPolicy, priceDimensions["pricingPolicy"])
	cblogger.Info("2.priceDemensionVal ", priceDemensionVal, hasPriceDimension, priceDimensionsKey)
	cblogger.Info("3.unitVal ", unitVal, hasUnit)

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
			cblogger.Info("filtered by priceDimension ", priceDemensionVal, priceDimensionsKey)
			return true
		}
	}

	if hasPricingPolicy {
		if pricingPolicyVal != "Reserved" {
			cblogger.Info("filtered by pricingPolicy reserved only ", pricingPolicyVal)
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
		if leaseContractLengthVal != termAttributes["LeaseContractLength"] { // 계약 기간 : 1yr, 3yr ...
			cblogger.Info("filtered by LeaseContractLength ", priceDemensionVal, termAttributes["LeaseContractLength"])
			return true
		}
	}

	if hasOfferingClass {
		if offeringClassVal != termAttributes["OfferingClass"] { // 계약 종류 : standard, convertible ...
			cblogger.Info("filtered by OfferingClass ", offeringClassVal, termAttributes["OfferingClass"])
			return true
		}
	}

	cblogger.Info("11.pricingPolicyVal ", pricingPolicyVal, hasPricingPolicy)
	cblogger.Info("22.priceDemensionVal ", priceDemensionVal, hasPriceDimension)
	cblogger.Info("33.unitVal ", unitVal, hasUnit)
	cblogger.Info("44.leaseContractLengthVal ", leaseContractLengthVal, hasLeaseContractLength)
	cblogger.Info("55.offeringClassVal ", offeringClassVal, hasOfferingClass)
	cblogger.Info("66.purchaseOptionVal ", purchaseOptionVal, hasPurchaseOption)

	return isFiltered
}
