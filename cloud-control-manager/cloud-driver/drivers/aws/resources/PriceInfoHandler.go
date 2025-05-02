package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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

func (priceInfoHandler *AwsPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	result := &irs.CloudPriceData{}
	result.Meta.Version = "v0.1"
	result.Meta.Description = "Multi-Cloud Price Info"

	priceMap := make(map[string]irs.Price)

	cblogger.Info("productFamily : ", productFamily)
	cblogger.Info("filter value : ", filterList)

	svc := priceInfoHandler.Client

	filters := []*pricing.Filter{}

	// add basic filters
	filters = append(filters, &pricing.Filter{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("marketoption"),
		Value: aws.String("OnDemand"),
	})

	filters = append(filters, &pricing.Filter{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("operatingSystem"),
		Value: aws.String("Linux"),
	})

	filters = append(filters, &pricing.Filter{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("tenancy"),
		Value: aws.String("Shared"),
	})

	filters = append(filters, &pricing.Filter{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("preInstalledSw"),
		Value: aws.String("NA"),
	})

	filters = append(filters, &pricing.Filter{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("capacitystatus"),
		Value: aws.String("Used"),
	})

	filters = append(filters, &pricing.Filter{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("currentGeneration"),
		Value: aws.String("Yes"),
	})

	if regionName != "" {
		filters = append(filters, &pricing.Filter{
			Type:  aws.String("EQUALS"),
			Field: aws.String("regionCode"),
			Value: aws.String(regionName),
		})
	} else {
		filters = append(filters, &pricing.Filter{
			Type:  aws.String("EQUALS"),
			Field: aws.String("regionCode"),
			Value: aws.String(priceInfoHandler.Region.Region),
		})
	}

	// Filter for productFamily, but not used because both 'Compute Instance' and 'Compute Instance (bare metal)' are needed
	// filters = append(filters, &pricing.Filter{
	// 	Type:  aws.String("EQUALS"),
	// 	Field: aws.String("productFamily"),
	// 	Value: aws.String(productFamily),
	// })

	// add user filters
	userFilters, err := setProductsInputRequestFilter(filterList)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}
	filters = append(filters, userFilters...)

	cblogger.Info("filters", filters)

	// set input parameters
	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters:     filters,
		MaxResults:  aws.Int64(100),
	}

	// proccess pagenation
	err = svc.GetProductsPages(input,
		func(page *pricing.GetProductsOutput, lastPage bool) bool {
			for _, awsPrice := range page.PriceList {
				productInfoMap := awsPrice["product"].(map[string]interface{})
				productFamilyVal, ok := productInfoMap["productFamily"].(string)
				if !ok {
					continue
				}

				// only productFamily: 'Compute Instance' & 'Compute Instance (bare metal)'
				if productFamilyVal != "Compute Instance" && productFamilyVal != "Compute Instance (bare metal)" {
					continue
				}
				productInfo, err := ExtractProductInfo(awsPrice, productFamilyVal)
				if err != nil {
					cblogger.Error(err)
					continue
				}

				// termsKey : OnDemand, Reserved
				for termsKey, termsValue := range awsPrice["terms"].(map[string]interface{}) {
					// only OnDemand
					if termsKey != "OnDemand" {
						continue
					}

					for _, policyValue := range termsValue.(map[string]interface{}) {
						priceDimensions := make(map[string]interface{})
						termAttributes := make(map[string]interface{})
						sku := ""

						if priceDimensionsVal, ok := policyValue.(map[string]interface{})["priceDimensions"]; ok {
							priceDimensions = priceDimensionsVal.(map[string]interface{})
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

						for priceDimensionsKey, priceDimensionsValue := range priceDimensions {
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

								// Parse price string to float
								priceStr := fmt.Sprintf("%s", val)
								priceFloat, err := strconv.ParseFloat(priceStr, 64)
								if err == nil {
									// Check decimal digits
									parts := strings.Split(priceStr, ".")
									decimalDigits := 0
									if len(parts) > 1 {
										// Calculate significant decimal digits (excluding trailing zeros)
										decimalDigits = len(strings.TrimRight(parts[1], "0"))
									}

									// For integer or less than 2 decimal digits
									if decimalDigits < 2 {
										// Format with 2 decimal places
										pricingPolicy.Price = fmt.Sprintf("%.2f", priceFloat)
									} else {
										// For 2 or more decimal digits, remove unnecessary zeros
										trimmedPrice := strings.TrimRight(fmt.Sprintf("%f", priceFloat), "0")
										// Remove trailing decimal point if exists
										if trimmedPrice[len(trimmedPrice)-1] == '.' {
											trimmedPrice = trimmedPrice[:len(trimmedPrice)-1]
										}
										pricingPolicy.Price = trimmedPrice
									}
								} else {
									pricingPolicy.Price = priceStr // Use original price string if parsing fails
								}

								// USD is Default.
								// if NO USD data, accept other currency.
								if key == "USD" {
									break
								}
							}

							// Convert 'Hrs' unit to 'Hour'
							unitStr := fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["unit"])
							if unitStr == "Hrs" {
								pricingPolicy.Unit = "Hour"
							} else {
								pricingPolicy.Unit = unitStr
							}

							// Extract and add policy
							aPrice, ok := AppendPolicyToPrice(priceMap, productInfo, pricingPolicy, awsPrice)
							if !ok {
								priceMap[productInfo.ProductId] = aPrice
							}
						}
					}
				}
			}
			return true // continue to the next page
		})

	if err != nil {
		cblogger.Error(err)
		return "", err
	}

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

// set filter for GetProductsInput
func setProductsInputRequestFilter(filterList []irs.KeyValue) ([]*pricing.Filter, error) {
	requestFilters := []*pricing.Filter{}

	if filterList != nil {
		for _, filter := range filterList {
			//--------------------- ProductInfo
			if filter.Key == "ProductId" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("sku"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "SpecName" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("instanceType"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "VCpu.Count" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("vcpu"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "MemSizeMiB" {
				filterValue := convertMiBtoGiBStringWithUnitForFilter(filter.Value)
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("memory"),
					Value: aws.String(filterValue),
				})
			}
			if filter.Key == "DiskSizeGB" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("storage"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "Gpu.Count" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("gpu"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "Gpu.MemSizeGB" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("gpuMemory"),
					Value: aws.String(filter.Value + " GB"),
				})
			}
			if filter.Key == "OSDistribution" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("operatingSystem"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "preInstalledSw" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("preInstalledSw"),
					Value: aws.String(filter.Value),
				})
			}
			//--------------------- PriceInfo
			if filter.Key == "PricingId" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("rateCode"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "PricingPolicy" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("terms"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "Unit" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("unit"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "Currency" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("pricePerUnit"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "Price" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("USD"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "LeaseContractLength" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("LeaseContractLength"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "OfferingClass" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("OfferingClass"),
					Value: aws.String(filter.Value),
				})
			}
			if filter.Key == "PurchaseOption" {
				requestFilters = append(requestFilters, &pricing.Filter{
					Type:  aws.String("TERM_MATCH"),
					Field: aws.String("PurchaseOption"),
					Value: aws.String(filter.Value),
				})
			}
		} //end of for
	} // end of if
	return requestFilters, nil
}

// extracts product information from the JSON value
func ExtractProductInfo(jsonValue aws.JSONValue, productFamily string) (irs.ProductInfo, error) {
	var productInfo irs.ProductInfo

	jsonString, err := json.MarshalIndent(jsonValue["product"].(map[string]interface{})["attributes"], "", "    ")
	if err != nil {
		cblogger.Error(err)
		return productInfo, err
	}
	switch productFamily {
	case "Compute Instance", "Compute Instance (bare metal)":
		err := setVMspecInfo(&productInfo, string(jsonString))
		if err != nil {
			return productInfo, err
		}
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
	productInfo.ZoneName = "NA" // AWS zone is Not Applicable - 202401
	productInfo.OSDistribution = fmt.Sprintf("%s", jsonValue["product"].(map[string]interface{})["attributes"].(map[string]interface{})["operatingSystem"])
	productInfo.Description = fmt.Sprintf("productFamily= %s, version= %s", jsonValue["product"].(map[string]interface{})["productFamily"], jsonValue["version"])
	productInfo.CSPProductInfo = jsonValue["product"]

	return productInfo, nil
}

func setVMspecInfo(productInfo *resources.ProductInfo, jsonValueString string) error {
	var jsonData map[string]string
	if err := json.Unmarshal([]byte(jsonValueString), &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	vcpu := jsonData["vcpu"]
	if vcpu == "" {
		return errors.New("missing required field: vcpu")
	}

	memoryInt := extractNumericValue(jsonData["memory"])

	instanceType := jsonData["instanceType"]
	if instanceType == "" {
		return errors.New("missing required field: instanceType")
	}

	regionCode := jsonData["regionCode"]
	if regionCode == "" {
		return errors.New("missing required field: regionCode")
	}

	// set GPU info if available
	var gpuInfo []irs.GpuInfo
	if gpuCount, ok := jsonData["gpu"]; ok && gpuCount != "0" {
		gpuCountInt, err := strconv.Atoi(gpuCount)
		if err != nil {
			return fmt.Errorf("failed to parse gpu: %w", err)
		}
		gpuMemoryFloat := extractNumericValue(jsonData["gpuMemory"])
		gpuInfo = []irs.GpuInfo{
			{
				Count:          gpuCount,
				Mfr:            "NA",
				Model:          "NA",
				MemSizeGB:      fmt.Sprintf("%d", int(gpuMemoryFloat)),
				TotalMemSizeGB: fmt.Sprintf("%d", int(float64(gpuCountInt)*gpuMemoryFloat)),
			},
		}
	}

	// set VMSpecInfo
	productInfo.VMSpecInfo = irs.VMSpecInfo{
		Region:     regionCode,
		Name:       instanceType,
		VCpu:       irs.VCpuInfo{Count: vcpu, ClockGHz: "-1"},
		MemSizeMiB: irs.ConvertGiBToMiBInt64(int64(memoryInt)),
		DiskSizeGB: "-1",
		Gpu:        gpuInfo,
	}

	return nil
}

func extractNumericValue(input string) float64 {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return -1
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return -1
	}
	return value
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

// "1024" (MiB) => "1 GiB"
func convertMiBtoGiBStringWithUnitForFilter(mibStr string) string {
	mibVal, err := strconv.ParseFloat(mibStr, 64)
	if err != nil {
		return mibStr
	}

	gibVal := mibVal / 1024
	if gibVal == float64(int64(gibVal)) {
		return fmt.Sprintf("%d GiB", int64(gibVal))
	}

	return fmt.Sprintf("%.1f GiB", gibVal)
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
