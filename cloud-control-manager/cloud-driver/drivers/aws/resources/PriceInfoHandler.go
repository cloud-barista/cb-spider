package resources

import (
	"encoding/json"
	"fmt"

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
// 3개 Region Endpoint에서만 Product 정보를 리턴합니다. getPricingClient에 Client *pricing.Pricing 정의
func (priceInfoHandler *AwsPriceInfoHandler) ListProductFamily(targetRegion string) ([]string, error) {
	var result []string
	input := &pricing.DescribeServicesInput{}
	for {
		services, err := priceInfoHandler.Client.DescribeServices(input)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
		for _, service := range services.Services {
			servicesTOString := fmt.Sprintf("%v", service)
			result = append(result, servicesTOString)
		}
		if services.NextToken != nil {
			input = &pricing.DescribeServicesInput{
				NextToken: services.NextToken,
			}
		} else {
			break
		}
	}
	return result, nil
}

func (priceInfoHandler *AwsPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {

	// describeServicesinput := &pricing.DescribeServicesInput{
	// 	ServiceCode: aws.String(productFamily),
	// 	MaxResults:  aws.Int64(1),
	// }
	// services, err := priceInfoHandler.Client.DescribeServices(describeServicesinput)
	// if services == nil {
	// 	cblogger.Error("No services in given productFamily. CHECK productFamily!")
	// 	return "", err
	// }
	// if err != nil {
	// 	cblogger.Error(err)
	// 	return "", err
	// }

	// var errstr string

	// getAttributeValuesinput := &pricing.GetAttributeValuesInput{}
	// for _, filter := range filterList {
	// 	if filter.Key == "AttributeName" {
	// 		for _, AttributeName := range services.Services[0].AttributeNames {
	// 			errstr = "No AttributeName in given productFamily. CHECK AttributeName!"
	// 			if *AttributeName == filter.Value {
	// 				errstr = ""
	// 				break
	// 			}
	// 		}
	// 		if errstr != "" {
	// 			cblogger.Error(errstr)
	// 			return "", nil
	// 		}
	// 		getAttributeValuesinput.AttributeName = aws.String(filter.Value)
	// 		break
	// 	}
	// }
	// getAttributeValuesinput.ServiceCode = aws.String(productFamily)

	// attributeValues, err := priceInfoHandler.Client.GetAttributeValues(getAttributeValuesinput)
	// if err != nil {
	// 	cblogger.Error(err)
	// 	return "", err
	// }

	// for _, filter := range filterList {
	// 	if filter.Key == "AttributeValue" {
	// 		for _, AttributeValue := range attributeValues.AttributeValues {
	// 			errstr = "No AttributeValue in given AttributeValue. CHECK AttributeValue!"
	// 			if *AttributeValue.Value == filter.Value {
	// 				errstr = ""
	// 				break
	// 			}
	// 		}
	// 		if errstr != "" {
	// 			cblogger.Error(errstr)
	// 			return "", nil
	// 		}
	// 		getAttributeValuesinput.AttributeName = aws.String(filter.Value)
	// 		break
	// 	}
	// }

	// getProductsinputfilter := &pricing.Filter{}
	// for _, filter := range filterList {
	// 	if filter.Key == "AttributeName" {
	// 		getProductsinputfilter.Field = aws.String(filter.Value)
	// 	} else if filter.Key == "Type" {
	// 		getProductsinputfilter.Type = aws.String(filter.Value)
	// 	} else if filter.Key == "AttributeValue" {
	// 		getProductsinputfilter.Value = aws.String(filter.Value)
	// 	}
	// }

	getProductsinputfilters := []*pricing.Filter{}
	var getProductsinputfilter pricing.Filter
	for _, filter := range filterList {
		err := json.Unmarshal([]byte(filter.Value), &getProductsinputfilter)
		getProductsinputfilters = append(getProductsinputfilters, &getProductsinputfilter)
		if err != nil {
			cblogger.Error(err)
			return "", err
		}
	}

	getProductsinputfilters = append(getProductsinputfilters, &pricing.Filter{
		Field: aws.String("regionCode"),
		Type:  aws.String("EQUALS"),
		Value: aws.String(regionName),
	})

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
	result.CloudPriceList = append(result.CloudPriceList, irs.CloudPrice{
		CloudName: "AWS",
	})

	// fmt.Println(priceinfos.PriceList)
	for _, price := range priceinfos.PriceList {
		var productInfo irs.ProductInfo

		jsonString, err := json.MarshalIndent(price["product"].(map[string]interface{})["attributes"], "", "    ")
		if err != nil {
			fmt.Println("Error:", err)
		}
		err = json.Unmarshal(jsonString, &productInfo)
		if err != nil {
			fmt.Println("Unmarshal Error:", err)
		}

		var jsonVal map[string]interface{}
		jsonData, err := json.MarshalIndent(price["terms"], "", "    ")
		if err != nil {
			fmt.Println("JSON 변환 오류:", err)
		}
		err = json.Unmarshal([]byte(jsonData), &jsonVal)
		if err != nil {
			fmt.Println("JSON 파싱 오류:", err)
		}

		productInfo.ProductId = fmt.Sprintf("%s", price["product"].(map[string]interface{})["sku"])
		productInfo.RegionName = fmt.Sprintf("%s", price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["regionCode"])
		productInfo.Description = fmt.Sprintf("productFamily %s, version %s", price["product"].(map[string]interface{})["productFamily"], price["version"])
		productInfo.CSPProductInfo = price["product"]
		awsJsonValtoMap(price)

		for termskey, _ := range price["terms"].(map[string]interface{}) {
			if termskey == "OnDemand" {
				for OnDemandkey, OnDemandvalue := range price["terms"].(map[string]interface{})["OnDemand"].(map[string]interface{}) {
					fmt.Println(OnDemandkey, OnDemandvalue)
				}
			}

		}

		// // var priceInfo irs.PriceInfo
		// // for _,term := range price["product"] {}
		// if price["terms"].(map[string]interface{})["OnDemand"] != nil {
		// 	fmt.Print(price["terms"].(map[string]interface{})["OnDemand"])

		// 	// for _, Key := range price["terms"].(map[string]irs.KeyValue{})["OnDemand"] {
		// 	// 	fmt.Println("Key", Key)
		// 	// }

		// 	// pricingPolicy := irs.PricingPolicies{}
		// 	// pricingPolicy.pricingPolicy = "OnDemand"
		// 	// pricingPolicy.PricingId =
		// 	// pricingPolicy.PricingId =
		// 	// pricingPolicy.PricingId =
		// 	// pricingPolicy.PricingId =
		// 	// pricingPolicy.PricingId =
		// }
		// fmt.Println(price["terms"].(map[string]interface{})["OnDemand"])
		// fmt.Println(price["terms"].(map[string]interface{})["Reserved"])

		break
	}

	resultString, err := json.Marshal(result)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	// result.CloudPriceList = append(result.CloudPriceList, irs.CloudPrice{})

	return string(resultString), nil
}

func awsJsonValtoMap(data aws.JSONValue) (map[string]interface{}, error) {
	var jsonVal map[string]interface{}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(jsonData), &jsonVal)
	if err != nil {
		return nil, err
	}
	return jsonVal, err
}
