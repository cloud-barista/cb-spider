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
// GetPriceInfo는 DescribeServices를 통해 옳바른 productFamily 인자만 검사합니다. -> AttributeName에 오류가 있을경우 빈값을 리턴
func (priceInfoHandler *AwsPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {

	describeServicesinput := &pricing.DescribeServicesInput{
		ServiceCode: aws.String(productFamily),
		MaxResults:  aws.Int64(1),
	}
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
			var getProductsinputfilter pricing.Filter
			err := json.Unmarshal([]byte(filter.Value), &getProductsinputfilter)
			getProductsinputfilters = append(getProductsinputfilters, &getProductsinputfilter)
			if err != nil {
				cblogger.Error(err)
				return "", err
			}
		}
	}
	if regionName != "" {
		getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
			Field: aws.String("regionCode"),
			Type:  aws.String("EQUALS"),
			Value: aws.String(regionName),
		})
	}

	getProductsinput := &pricing.GetProductsInput{
		Filters:     getProductsinputfilters,
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
							pricingPolicy.Unit = fmt.Sprintf("%s", priceDimensionsValue.(map[string]interface{})["unit"])
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
