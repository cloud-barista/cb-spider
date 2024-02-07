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
	priceMap := make(map[string]irs.Price)
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
	getProductsinputfilters := []*pricing.Filter{}

	if filterList != nil {
		for _, filter := range filterList {
			if filter.Key == "instanceType" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("instanceType"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}

			if filter.Key == "operatingSystem" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("operatingSystem"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "vcpu" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("vcpu"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "productId" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("sku"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "memory" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("memory"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "storage" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("storage"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "gpu" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("gpu"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "gpuMemory" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("gpuMemory"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "preInstalledSw" {
				getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
					Field: aws.String("preInstalledSw"),
					Type:  aws.String("TERM_MATCH"),
					Value: aws.String(filter.Value),
				})
			}
			// if filter.Key == "leaseContractLength" {
			// 	getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
			// 		Field: aws.String("leaseContractLength"),
			// 		Type:  aws.String("TERM_MATCH"),
			// 		Value: aws.String(filter.Value),
			// 	})
			// }

		}
	}

	// filter조건에 region 지정.
	if regionName != "" {
		getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
			Field: aws.String("regionCode"),
			Type:  aws.String("EQUALS"),
			Value: aws.String(regionName),
		})
	} else {
		getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
			Field: aws.String("regionCode"),
			Type:  aws.String("EQUALS"),
			Value: aws.String(priceInfoHandler.Region.Region),
		})
	}

	getProductsinput := &pricing.GetProductsInput{
		Filters:     getProductsinputfilters,
		ServiceCode: aws.String(productFamily),
	}
	cblogger.Info("get Products request", getProductsinput)
	priceinfos, err := priceInfoHandler.Client.GetProducts(getProductsinput)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}
	cblogger.Info("get Products response", priceinfos)

	result := &irs.CloudPriceData{}
	result.Meta.Version = "v0.1"
	result.Meta.Description = "Multi-Cloud Price Info"
	// for the test
	// cblogger.Info("productInfo", priceinfos)
	for _, price := range priceinfos.PriceList {
		cblogger.Info("=-=-=-=-=-=-=-=-", price)
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

		productId := fmt.Sprintf("%s", price["product"].(map[string]interface{})["sku"])
		productInfo.ProductId = fmt.Sprintf("%s", price["product"].(map[string]interface{})["sku"])
		productInfo.RegionName = fmt.Sprintf("%s", price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["regionCode"])
		productInfo.Description = fmt.Sprintf("productFamily= %s, version= %s", price["product"].(map[string]interface{})["productFamily"], price["version"])
		productInfo.CSPProductInfo = price["product"]
		productInfo.ZoneName = "NA" // AWS zone is Not Applicable - 202401

		// var priceInfo irs.PriceInfo
		// priceInfo.CSPPriceInfo = price["terms"]
		// cblogger.Info("priceInfo.CSPPriceInfo******************** = ", priceInfo.CSPPriceInfo)
		// cblogger.Info("priceInfo.CSPPriceInfo^^^^^^^^^^^^^^^^^^^^ = ", priceInfo)
		// termsKey : OnDemand, Reserved
		for termsKey, termsValue := range price["terms"].(map[string]interface{}) {
			cblogger.Info("now termsKey = ", termsKey)
			hasPricingPolicyVal := false
			pricingPolicyVal := ""

			hasPriceDimension := false
			priceDemensionVal := ""

			hasunit := false
			unitVal := ""

			hasLeaseContractLength := false
			LeaseContractLengthVal := ""

			hasOfferingClass := false
			OfferingClassVal := ""

			hasPurchaseOption := false
			PurchaseOptionVal := ""

			if filterList != nil {

				for _, filter := range filterList {
					// find filter conditions
					if filter.Key == "pricingPolicy" {
						hasPricingPolicyVal = true
						pricingPolicyVal = filter.Value
						continue
					}

					if filter.Key == "pricingId" {
						hasPriceDimension = true
						priceDemensionVal = filter.Value
						continue
					}
					if filter.Key == "unit" {
						hasunit = true
						unitVal = filter.Value
						continue
					}
					if filter.Key == "LeaseContractLength" {
						hasLeaseContractLength = true
						LeaseContractLengthVal = filter.Value
						continue
					}
					if filter.Key == "OfferingClass" {
						hasOfferingClass = true
						OfferingClassVal = filter.Value
						continue
					}
					if filter.Key == "PurchaseOption" {
						hasPurchaseOption = true
						PurchaseOptionVal = filter.Value
						continue
					}
				}
				// check filters
				if hasPricingPolicyVal && pricingPolicyVal != termsKey {
					cblogger.Info("filtered by pricingPolicy ", pricingPolicyVal, termsKey)
					continue
				}
			}

			for _, policyValue := range termsValue.(map[string]interface{}) {
				//cblogger.Info("termsValue(((((((", termsValue) // OnDemand 밑 map
				var pricingPolicy irs.PricingPolicies
				for innerpolicyKey, innerpolicyValue := range policyValue.(map[string]interface{}) {
					//cblogger.Info("policyvalue %%%%%%%%%%%%", policyvalue.(map[string]interface{})) // here
					//cblogger.Info("innerpolicyKey ??????????", innerpolicyKey)                      // termAttribute
					//cblogger.Info("innerpolicyValue !!!!!!!!!!", innerpolicyValue)                  // map

					if innerpolicyKey == "priceDimensions" {
						filterResult := false // true면 filter 통과 된 것임.
						for priceDimensionsKey, priceDimensionsValue := range innerpolicyValue.(map[string]interface{}) {
							//cblogger.Info("priceDimensionsValue))))))))))))", priceDimensionsValue)
							if filterList != nil {
								// check filters
								if hasPriceDimension && priceDemensionVal != priceDimensionsKey {
									cblogger.Info("filtered by priceDimensions ", priceDemensionVal, priceDimensionsKey)
									continue
								}
								//pricingId의 unit값이 필터 값으로 들어오면 unit 값을 받은 값으로 설정
								foundSku := false
								for _, skukey := range priceDimensionsValue.(map[string]interface{}) {
									// check filters
									if hasunit && unitVal == skukey {
										foundSku = true
										break
									}
								}
								if hasunit && !foundSku { // sku를 못 찾았으면 skip.
									cblogger.Info("filtered by Sku ", hasunit, foundSku)
									continue
								}
							}

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
								// check filters
								// if hasTerm && termVal != termsKey {
								// 	cblogger.Info("filtered by pricingPolicy ", termVal,  termsKey)
								// 	continue
								// }
							}
							pricingPolicy.Unit = fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["unit"])

							// 기존 로직에서 price add하던 부분
							// 여기까지 왔으면 filterResult를 완료(true)로 바꿈
							filterResult = true

						} // end of for
						if !filterResult {
							continue // filter걸린게 있으면 filterResult가 false로 유지되었을 것임
						}
					}

					// leaseContractLength, offeringClass,purchaseOption 필터 추가
					// onDemand는 termAttributes 가 비어있을 수 있음.
					//aPrice, ok := priceMap[productId]

					if innerpolicyKey == "termAttributes" {
						filterResult := false // true면 filter 통과 된 것임.
						cblogger.Info("terms ::: ", termsKey)
						cblogger.Info("termAttribute ::: ", innerpolicyValue)
						for termAttributeskey, termAttributesValue := range innerpolicyValue.(map[string]interface{}) {
							cblogger.Info("termAttributeskey!********", termAttributeskey)
							cblogger.Info("termAttributesValue2////////", termAttributesValue) // TODO : map이 아니라 string값임.
							if filterList != nil {
								foundSku := false
								cblogger.Info("go sku ", termAttributesValue)
								for _, skukey := range termAttributesValue.(map[string]interface{}) {
									cblogger.Info("skukey ", skukey)
									// check filters
									if hasLeaseContractLength && LeaseContractLengthVal == skukey {
										foundSku = true
										break
									}
									if hasOfferingClass && OfferingClassVal == skukey {
										foundSku = true
										break
									}
									if hasPurchaseOption && PurchaseOptionVal == skukey {
										foundSku = true
										break
									}
									cblogger.Info("end skukey ", skukey)

								}
								if hasLeaseContractLength && !foundSku { // sku를 못 찾았으면 skip.
									cblogger.Info("filtered by Sku ", hasLeaseContractLength, foundSku)
									continue
								}
								if hasOfferingClass && !foundSku { // sku를 못 찾았으면 skip.
									cblogger.Info("filtered by Sku ", hasOfferingClass, foundSku)
									continue
								}
								if hasPurchaseOption && !foundSku { // sku를 못 찾았으면 skip.
									cblogger.Info("filtered by Sku ", hasPurchaseOption, foundSku)
									continue
								}
							}
							cblogger.Info("set leaseContractLength ", termAttributesValue.(map[string]interface{})["LeaseContractLength"])
							// 		spew.Dump("termAttributesValue.(map[string]interface{})[LeaseContractLength]5555555555", termAttributesValue.(map[string]interface{})["LeaseContractLength"])
							// 		spew.Dump("termAttributesValue6666666666", termAttributesValue)
							pricingPolicy.PricingPolicyInfo.LeaseContractLength = fmt.Sprintf("%s", termAttributesValue.(map[string]interface{})["LeaseContractLength"])
							pricingPolicy.PricingPolicyInfo.OfferingClass = fmt.Sprint("%s", innerpolicyValue.(map[string]interface{})["OfferingClass"])
							pricingPolicy.PricingPolicyInfo.PurchaseOption = fmt.Sprint("%s", innerpolicyValue.(map[string]interface{})["PurchaseOption"])

							filterResult = true // 여기까지 왔으면 filterResult를 완료(true)로 바꿈
						}
						cblogger.Info("filterResult2 ", filterResult)
						if !filterResult {
							continue // filter걸린게 있으면 filterResult가 false로 유지되었을 것임
						}
						cblogger.Info("LeaseContractLength !@!@!@!@!@", pricingPolicy.PricingPolicyInfo.LeaseContractLength)
					}

					aPrice, ok := priceMap[productId]

					if ok { // product가 존재하면 policy 추가
						cblogger.Info("product exist ", productId)
						aPrice.PriceInfo.PricingPolicies = append(aPrice.PriceInfo.PricingPolicies, pricingPolicy)
						// aPrice.PriceInfo.CSPPriceInfo = append(aPrice.PriceInfo.CSPPriceInfo.([]string), cspPriceInfo...)
						// var priceInfo irs.PriceInfo
						// priceInfo.CSPPriceInfo = price["terms"]
						priceMap[productId] = aPrice

					} else { // product가 없으면 price 추가
						cblogger.Info("product not exist ", productId)

						newPriceInfo := irs.PriceInfo{}
						newPolicies := []irs.PricingPolicies{}
						newPolicies = append(newPolicies, pricingPolicy)

						newPriceInfo.PricingPolicies = newPolicies
						newPriceInfo.CSPPriceInfo = price["terms"] // 새로운 가격이면 terms아래값을 넣는다.

						// newCSPPriceInfo := []string{}
						// newCSPPriceInfo = append(newCSPPriceInfo, priceResponseStr)
						// newPriceInfo.CSPPriceInfo = newCSPPriceIn
						newPrice := irs.Price{}
						newPrice.PriceInfo = newPriceInfo
						newPrice.ProductInfo = productInfo

						priceMap[productId] = newPrice
					}

				}
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
