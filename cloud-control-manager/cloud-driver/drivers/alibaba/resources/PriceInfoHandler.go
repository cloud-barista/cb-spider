// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"encoding/json"
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	bssopenapi "github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi" // update to v1.62.327 from v1.61.1743, due to QuerySkuPriceListRequest struct

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaPriceInfoHandler struct {
	BssClient *bssopenapi.Client
}

type ProductInfo struct {
	diskType    string
	diskMinSize int64
	diskMaxSize int64
	unit        string
}

func (priceInfoHandler *AlibabaPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	// productListresponse, err := QueryProductList(priceInfoHandler.BssClient)
	// if err != nil {
	// 	return nil, err
	// }

	// var familyList []string
	// for _, Product := range productListresponse.Data.ProductList.Product {
	// 	familyList = append(familyList, Product.ProductCode)
	// }

	// 컴퓨팅 인프라 서비스를 호출하기 위한 별도 API 존재하지 않음.
	familyList := []string{
		"ecs",
	}

	return familyList, nil
}

func (priceInfoHandler *AlibabaPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter []irs.KeyValue) (string, error) {
	// 사용자가 입력한 productFamily의 구독 타입을 얻기위해 전체 서비스 호출하여 확인
	productListresponse, err := QueryProductList(priceInfoHandler.BssClient)
	if err != nil {
		return "", err
	}
	targetProducts := []bssopenapi.Product{}
	for _, product := range productListresponse.Data.ProductList.Product {
		if product.ProductCode == productFamily {
			targetProducts = append(targetProducts, product)
		}
	}
	cblogger.Info(targetProducts)

	if len(targetProducts) == 0 {
		cblogger.Errorf("There is no match productFamily input - [%s]", productFamily)
		return "", err
	}

	CloudPriceData := irs.CloudPriceData{}
	CloudPriceData.Meta.Version = "v0.1"
	CloudPriceData.Meta.Description = "Multi-Cloud Price Info"

	for _, product := range targetProducts {

		CloudPrice := irs.CloudPrice{}
		CloudPrice.CloudName = "Alibaba"
		if product.SubscriptionType == "PayAsYouGo" {
			pricingModuleRequest := bssopenapi.CreateDescribePricingModuleRequest()
			pricingModuleRequest.Scheme = "https"
			pricingModuleRequest.SubscriptionType = product.SubscriptionType
			pricingModuleRequest.ProductCode = product.ProductCode
			pricingModuleRequest.ProductType = product.ProductType
			pricingModulesPayAsYouGo, err := priceInfoHandler.BssClient.DescribePricingModule(pricingModuleRequest)
			if err != nil {
				cblogger.Error(err)
				return "", err
			}
			isExist := bool(false)
			// var pricingModuleCurrency string
			var pricingModulePriceType string
			for _, pricingModule := range pricingModulesPayAsYouGo.Data.ModuleList.Module {
				if pricingModule.ModuleCode == "InstanceType" {
					for _, config := range pricingModule.ConfigList.ConfigList {
						if config == "InstanceType" {
							// pricingModuleCurrency = pricingModule.Currency
							// fmt.Println(pricingModuleCurrency)
							pricingModulePriceType = pricingModule.PriceType
							isExist = true
							break
						}
					}
					break
				}
			}

			if !isExist {
				cblogger.Errorf("There is no InstanceType Module Config- [%s]", productFamily)
				return "", err
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
			getPayAsYouGoPriceRequest.QueryParams["Region"] = regionName // 리전별 응답에 차이가 없는 것 같음.

			for _, pricingModulesAttr := range pricingModulesPayAsYouGo.Data.AttributeList.Attribute {
				if pricingModulesAttr.Code == "InstanceType" {
					for _, attr := range pricingModulesAttr.Values.AttributeValue {
						// ModuleList.n.XXXXXX 에서 키 값의 인덱스를 1 부터 50까지 넣을 수 있지만
						// 특정 InstanceType에 Attr 에 존재하지만, 쿼리에 InternalError 발생.
						// 부득이 단건 호출하여 오류 처리함. -> 쿼타 테스트에 문제 발견되지 않음.(초당 10회 제한)
						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.ModuleCode"] = "InstanceType"
						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.PriceType"] = pricingModulePriceType
						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.Config"] = "InstanceType:" + attr.Value
						priceResponse, err := priceInfoHandler.BssClient.ProcessCommonRequest(getPayAsYouGoPriceRequest)
						if err != nil {
							cblogger.Errorf(err.Error())
						} else {
							var priceResp map[string]interface{}
							err := json.Unmarshal([]byte(priceResponse.GetHttpContentString()), &priceResp)
							if err != nil {
								cblogger.Errorf("Error parsing [%s] JSON:%s", attr.Value, err.Error())
							} else if priceResp["Code"] != "Success" {
								cblogger.Errorf("[%s] ErrCode : %s", attr.Value, priceResp["Code"])
							} else {
								DescribeInstanceRequest := requests.NewCommonRequest()
								DescribeInstanceRequest.Method = "POST"
								DescribeInstanceRequest.Scheme = "https" // https | http
								DescribeInstanceRequest.Domain = "ecs.ap-southeast-1.aliyuncs.com"
								DescribeInstanceRequest.Version = "2014-05-26"
								DescribeInstanceRequest.ApiName = "DescribeInstanceTypes"
								DescribeInstanceRequest.QueryParams["InstanceTypes.1"] = attr.Value
								instanceResponse, err := priceInfoHandler.BssClient.ProcessCommonRequest(DescribeInstanceRequest)
								if err != nil {
									cblogger.Error(err.Error())
								} else {
									var instanceResp map[string]interface{}
									err = json.Unmarshal([]byte(instanceResponse.GetHttpContentString()), &instanceResp)
									if err != nil {
										cblogger.Errorf("Error parsing JSON:%s", err.Error())
									} else {
										instanceTypeArray, valid := instanceResp["InstanceTypes"].(map[string]interface{})["InstanceType"].([]interface{})
										if valid && len(instanceTypeArray) > 0 {
											PriceList := irs.PriceList{}
											PriceList.ProductInfo.CSPProductInfo = instanceResponse.GetHttpContentString()
											PriceList.ProductInfo.ProductId = "NA"
											PriceList.ProductInfo.RegionName = "NA"
											PriceList.ProductInfo.InstanceType = fmt.Sprintf("%s", instanceTypeArray[0].(map[string]interface{})["InstanceTypeId"])
											PriceList.ProductInfo.Vcpu = fmt.Sprintf("%s", instanceTypeArray[0].(map[string]interface{})["CpuCoreCount"])
											PriceList.ProductInfo.Memory = fmt.Sprintf("%s", instanceTypeArray[0].(map[string]interface{})["MemorySize"])
											PriceList.ProductInfo.Storage = fmt.Sprintf("%s", instanceTypeArray[0].(map[string]interface{})["LocalStorageCategory"])
											PriceList.ProductInfo.Gpu = fmt.Sprintf("%s", instanceTypeArray[0].(map[string]interface{})["GPUSpec"])
											PriceList.ProductInfo.GpuMemory = fmt.Sprintf("%s", instanceTypeArray[0].(map[string]interface{})["GPUAmount"])
											PriceList.ProductInfo.OperatingSystem = "NA"
											PriceList.ProductInfo.PreInstalledSw = "NA"
											PriceList.ProductInfo.VolumeType = "NA"
											PriceList.ProductInfo.StorageMedia = "NA"
											PriceList.ProductInfo.MaxVolumeSize = "NA"
											PriceList.ProductInfo.MaxIOPSVolume = "NA"
											PriceList.ProductInfo.MaxThroughputVolume = "NA"

											priceInfo := irs.PriceInfo{}
											priceInfo.CSPPriceInfo = priceResponse.GetHttpContentString()
											pricingPolicy := irs.PricingPolicies{}
											pricingPolicy.PricingId = "NA"
											pricingPolicy.PricingPolicy = product.SubscriptionType
											pricingPolicy.Unit = pricingModulePriceType
											pricingPolicy.Currency = fmt.Sprintf("%s", priceResp["Data"].(map[string]interface{})["Currency"])
											pricingPolicy.Price = fmt.Sprintf("%f", priceResp["Data"].(map[string]interface{})["ModuleDetails"].(map[string]interface{})["ModuleDetail"].([]interface{})[0].(map[string]interface{})["OriginalCost"])
											pricingPolicy.Description = "NA"
											pricingPolicy.PricingPolicyInfo = &irs.PricingPolicyInfo{
												LeaseContractLength: "NA",
												OfferingClass:       "NA",
												PurchaseOption:      "NA",
											}
											priceInfo.PricingPolicies = append(priceInfo.PricingPolicies, pricingPolicy)

											PriceList.PriceInfo = priceInfo

											CloudPrice.PriceList = append(CloudPrice.PriceList, PriceList)

										} else {
											cblogger.Errorf("[%s] instanceType is Empty", attr.Value)
										}

										CloudPriceData.CloudPriceList = append(CloudPriceData.CloudPriceList, CloudPrice)

									}
								}

							}
						}
					}
				}
			}
		} else if product.SubscriptionType == "Subscription" {
			fmt.Println("NOT IMPLEMENT")
		}
	}

	resultString, err := json.Marshal(CloudPriceData)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return string(resultString), nil
}

/////// 이전 드라이버 개발

// func (priceInfoHandler *AlibabaPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter []irs.KeyValue) (string, error) {

// 	// sequence
// 	// QueryProductList (ProductCode) ->
// 	// (ProductCode) QueryCommodityListView (CommodityCode) ->
// 	// (CommodityCode) QueryPriceEntityListView (PriceFactorCode,PriceFactorValueList , PriceEntityCode) ->
// 	// (CommodityCode, PriceFactorCode, PriceFactorValueList, PriceEntityCode) QuerySkuPriceListViewPagination -> result

// 	querySkuPriceListRequestVar := bssopenapi.CreateQuerySkuPriceListRequest()
// 	querySkuPriceListRequestVar.PageSize = requests.NewInteger(20) // recommand 20 at one request
// 	querySkuPriceListRequestVar.Scheme = "https"
// 	// querySkuPriceListRequestVar.Lang = "en" // language struct is not exist
// 	// querySkuPriceListRequestVar.Language = "en"
// 	asd := bssopenapi.QuerySkuPriceListPriceFactorConditionMap{

// 	}
// 	querySkuPriceListRequestVar.CommodityCode = productFamily

// 	// option filter is not working
// 	querySkuPriceListRequestVar.PriceFactorConditionMap = make(map[string]*[]string)

// 	// option filter is not working
// 	// region code is not same ex) vm_region_no
// 	// querySkuPriceListRequestVar.PriceFactorConditionMap[""] = &regionName

// 	for _, keyValue := range filter {
// 		if keyValue.Key == "PriceEntityCode" {
// 			querySkuPriceListRequestVar.PriceEntityCode = keyValue.Value
// 		} else {
// 			// option filter is not working
// 			values := strings.Split(keyValue.Value, ",")
// 			querySkuPriceListRequestVar.PriceFactorConditionMap[keyValue.Key] = &values
// 		}
// 	}

// 	fmt.Println("Req Var###############################")
// 	spew.Dump(querySkuPriceListRequestVar)
// 	fmt.Println(querySkuPriceListRequestVar)
// 	fmt.Println("Req Var###############################")

// 	var priceList []bssopenapi.SkuPricePageDTO
// 	for {
// 		response, err := priceInfoHandler.BssClient.QuerySkuPriceList(querySkuPriceListRequestVar)
// 		fmt.Println(response)
// 		if err != nil {
// 			cblogger.Error(err)
// 		}
// 		priceList = append(priceList, response.Data.SkuPricePage.SkuPriceList...)
// 		if response.Data.SkuPricePage.NextPageToken != "" {
// 			querySkuPriceListRequestVar.NextPageToken = response.Data.SkuPricePage.NextPageToken
// 		} else {
// 			break
// 		}

// 		break
// 	}

// 	fmt.Println(&priceList)
// 	fmt.Println(priceList)

// 	return "priceInfo", nil
// }
