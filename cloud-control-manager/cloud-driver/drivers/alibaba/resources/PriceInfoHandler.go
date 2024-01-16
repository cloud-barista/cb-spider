// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"encoding/json"
	"fmt"
	"strconv"

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
type PriceInfoAli struct {
	Data Data   `json:"Data"`
	Code string `json:"Code"`
}

// Alibaba GetSubscriptionPrice ModuleDetails, PromotionDetails 배열 데이터 존재
type Data struct {
	OriginalPrice    float64
	DiscountPrice    float64
	Currency         string `json:"Currency,omitempty"`
	Quantity         int64
	ModuleDetails    ModuleDetails    `json:"ModuleDetails"`
	PromotionDetails PromotionDetails `json:"PromotionDetails"`
	TradePrice       float64
}
type ModuleDetails struct {
	ModuleDetail []ModuleDetail `json:"ModuleDetail"`
}

type ModuleDetail struct {
	UnitPrice         float64
	ModuleCode        string
	CostAfterDiscount float64
	Price             float64 `json:"OriginalCost"`
	InvoiceDiscount   float64
}

// Alibaba ServicePeriodUnit par 값을 Year로 넘길시 프로모션 관련 정보도 리턴을 해줌.
type PromotionDetails struct {
	PromotionDetail []PromotionDetail
}

type PromotionDetail struct {
	PromotionName string `json:"PromotionName,omitempty"`
	PromotionId   int64  `json:"PromotionId,omitempty"`
}

// Alibaba GetSubscriptionPrice end

// Alibaba DescribeInstanceTypes Response start
type ProudctInstanceTypes struct {
	InstanceTypes InstanceTypes `json:"InstanceTypes"`
}
type InstanceTypes struct {
	InstanceType []InstanceType `json:"InstanceType"`
}
type InstanceType struct {
	InstanceType string  `json:"InstanceTypeId,omitempty"`
	Vcpu         int64   `json:"CpuCoreCount,omitempty"`
	Memory       float64 `json:"MemorySize,omitempty"`
	Storage      string  `json:"LocalStorageCategory,omitempty"`
	Gpu          string  `json:"GPUSpec,omitempty"`
	GpuMemory    int64   `json:"GPUAmount,omitempty"`
}

type InstanceTypesResponse struct {
	RequestId     string `json:"RequestId"`
	NextToken     string `json:"NextToken"`
	InstanceTypes struct {
		InstanceType []ecs.InstanceType `json:"InstanceType"`
	} `json:"InstanceTypes"`
}

func (priceInfoHandler *AlibabaPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	productListresponse, err := QueryProductList(priceInfoHandler.BssClient)
	if err != nil {
		return nil, err
	}

	var familyList []string
	for _, Product := range productListresponse.Data.ProductList.Product {
		familyList = append(familyList, Product.ProductCode)
	}

	return familyList, nil
}

func (priceInfoHandler *AlibabaPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter []irs.KeyValue) (string, error) {
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
		// product 별 pricing종류 [PayAsYouGo, Subscription]
		if product.SubscriptionType == "PayAsYouGo" {
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

			isExist := bool(false)
			var pricingModulePriceType string
			for _, pricingModule := range pricingModulesPayAsYouGo.Data.ModuleList.Module {
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
				break
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
						cblogger.Infof("Now query is : [%s] %s ", product.SubscriptionType, attr.Value)
						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.ModuleCode"] = "InstanceType"
						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.PriceType"] = pricingModulePriceType // Hour
						getPayAsYouGoPriceRequest.QueryParams["ModuleList.1.Config"] = "InstanceType:" + attr.Value

						priceResponse, err := priceInfoHandler.BssClient.ProcessCommonRequest(getPayAsYouGoPriceRequest)
						if err != nil {
							cblogger.Error(err.Error())
							continue
						}

						pricingPolicy, priceResponseStr, err := BindpricingPolicy(priceResponse.GetHttpContentString(), product.SubscriptionType, pricingModulePriceType, regionName, attr.Value)
						if err != nil {
							cblogger.Error(err.Error())
							continue
						}

						productId := regionName + "_" + attr.Value
						existProduct := false
						for idx, aPriceList := range cloudPrice.PriceList {
							productInfo := &aPriceList.ProductInfo
							if productInfo.ProductId == productId { // 동일한  product가 있으면 policy만 추가한다.
								aPriceList.PriceInfo.PricingPolicies = append(aPriceList.PriceInfo.PricingPolicies, pricingPolicy)
								aPriceList.PriceInfo.CSPPriceInfo = append(aPriceList.PriceInfo.CSPPriceInfo.([]string), priceResponseStr)
								cloudPrice.PriceList[idx] = aPriceList
								existProduct = true
								break
							}
						}

						if !existProduct { // product가 없으면 조회해서 추가
							newProductInfo, err := GetDescribeInstanceTypesForPricing(priceInfoHandler.BssClient, regionName, attr.Value)
							if err != nil {
								cblogger.Errorf("[%s] instanceType is Empty", attr.Value)
								continue
							}
							newPrice := irs.Price{}
							newPrice.ProductInfo = newProductInfo
							newPriceInfo := irs.PriceInfo{}
							newPriceInfo.PricingPolicies = append(newPriceInfo.PricingPolicies, pricingPolicy)

							newCSPPriceInfo := []string{}
							newCSPPriceInfo = append(newCSPPriceInfo, priceResponseStr)
							newPriceInfo.CSPPriceInfo = newCSPPriceInfo
							newPrice.PriceInfo = newPriceInfo // priceList 를 돌면서 priceInfo 안의  productID가 같은 것 추출
							cloudPrice.PriceList = append(cloudPrice.PriceList, newPrice)

						}
					}
				}
			}
		} else if product.SubscriptionType == "Subscription" {

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
							cblogger.Infof("Now query is : [%s] %s ", product.SubscriptionType, attr.Value)

							// 특정 InstanceType에 Attr 에 존재하지만, 쿼리에 InternalError 발생. 부득이 단건 호출.
							getSubscriptionPrice.QueryParams["ModuleList.1.ModuleCode"] = "InstanceType"
							getSubscriptionPrice.QueryParams["ModuleList.1.Config"] = "InstanceType:" + attr.Value

							priceResponse, err := priceInfoHandler.BssClient.ProcessCommonRequest(getSubscriptionPrice)
							if err != nil {
								cblogger.Error(err.Error())
								continue
							}

							pricingPolicy, priceResponseStr, err := BindpricingPolicy(priceResponse.GetHttpContentString(), product.SubscriptionType, pricingModulePriceType, regionName, attr.Value)
							if err != nil {
								cblogger.Error(err.Error())
								continue
							}
							productId := regionName + "_" + attr.Value
							existProduct := false
							for idx, aPriceList := range cloudPrice.PriceList {
								productInfo := &aPriceList.ProductInfo
								if productInfo.ProductId == productId { // 동일한  product가 있으면 policy만 추가한다.
									if aPriceList.PriceInfo.PricingPolicies != nil {
										aPriceList.PriceInfo.PricingPolicies = append(aPriceList.PriceInfo.PricingPolicies, pricingPolicy)

									} else {
										newPricingPolicies := []irs.PricingPolicies{}
										newPricingPolicies = append(newPricingPolicies, pricingPolicy)

										aPriceList.PriceInfo.PricingPolicies = newPricingPolicies
									}
									if aPriceList.PriceInfo.CSPPriceInfo != nil {
										aPriceList.PriceInfo.CSPPriceInfo = append(aPriceList.PriceInfo.CSPPriceInfo.([]string), priceResponseStr)

									} else {
										newCSPPriceInfo := []string{}
										newCSPPriceInfo = append(newCSPPriceInfo, priceResponseStr)
										aPriceList.PriceInfo.CSPPriceInfo = newCSPPriceInfo

									}

									cloudPrice.PriceList[idx] = aPriceList
									existProduct = true
									break
								}
							}

							if !existProduct { // product가 없으면 조회해서 추가
								newProductInfo, err := GetDescribeInstanceTypesForPricing(priceInfoHandler.BssClient, regionName, attr.Value)
								if err != nil {
									cblogger.Errorf("[%s] instanceType is Empty", attr.Value)
									continue
								}
								newPrice := irs.Price{}
								newPrice.ProductInfo = newProductInfo
								newPriceInfo := irs.PriceInfo{}
								newPricingPolicies := []irs.PricingPolicies{}
								newPricingPolicies = append(newPricingPolicies, pricingPolicy)

								newCSPPriceInfo := []string{}
								newCSPPriceInfo = append(newCSPPriceInfo, priceResponseStr)

								newPriceInfo.PricingPolicies = newPricingPolicies
								newPriceInfo.CSPPriceInfo = newCSPPriceInfo

								newPrice.PriceInfo = newPriceInfo // priceList 를 돌면서 priceInfo 안의  productID가 같은 것 추출

								cloudPrice.PriceList = append(cloudPrice.PriceList, newPrice)
							}
						}
					}
				}
			}
		}

	}

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
func GetDescribeInstanceTypesForPricing(bssClient *bssopenapi.Client, regionName string, instanceType string) (irs.ProductInfo, error) {
	DescribeInstanceRequest := requests.NewCommonRequest()
	DescribeInstanceRequest.Method = "POST"
	DescribeInstanceRequest.Scheme = "https" // https | http
	DescribeInstanceRequest.Domain = "ecs.ap-southeast-1.aliyuncs.com"
	DescribeInstanceRequest.Version = "2014-05-26"
	DescribeInstanceRequest.ApiName = "DescribeInstanceTypes"
	DescribeInstanceRequest.QueryParams["InstanceTypes.1"] = instanceType

	productInfo := irs.ProductInfo{}
	instanceResponse, err := bssClient.ProcessCommonRequest(DescribeInstanceRequest)
	if err != nil {
		cblogger.Error(err.Error())
		return productInfo, err
	}
	instanceResp := ProudctInstanceTypes{}
	err = json.Unmarshal([]byte(instanceResponse.GetHttpContentString()), &instanceResp)
	if err != nil {
		cblogger.Errorf("Error parsing JSON:%s", err.Error())
	} else {
		if len(instanceResp.InstanceTypes.InstanceType) > 0 {
			resultProduct := instanceResp.InstanceTypes.InstanceType[0]
			productInfo.CSPProductInfo = instanceResponse.GetHttpContentString()
			productInfo.ProductId = regionName + "_" + instanceType
			productInfo.RegionName = regionName
			productInfo.ZoneName = "NA"
			productInfo.InstanceType = resultProduct.InstanceType
			productInfo.Vcpu = strconv.FormatInt(resultProduct.Vcpu, 10)
			productInfo.Memory = strconv.FormatFloat(resultProduct.Memory, 'f', -1, 64)
			productInfo.Storage = resultProduct.Storage
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
		}
	}

	return productInfo, nil
}

func BindpricingPolicy(priceResponse string, subscriptionType string, pricingModulePriceType string, regionName string, instanceType string) (irs.PricingPolicies, string, error) {
	priceResp := PriceInfoAli{}

	err := json.Unmarshal([]byte(priceResponse), &priceResp)
	if err != nil {
		return irs.PricingPolicies{}, "", err
	}

	pricingPolicy := irs.PricingPolicies{}
	pricingPolicy.PricingId = regionName + "_" + instanceType + "_" + subscriptionType + "_" + pricingModulePriceType //"NA"
	pricingPolicy.PricingPolicy = subscriptionType
	pricingPolicy.Unit = pricingModulePriceType
	pricingPolicy.Currency = priceResp.Data.Currency
	if len(priceResp.Data.ModuleDetails.ModuleDetail) > 0 {
		resultModuleDetailPrice := priceResp.Data.ModuleDetails.ModuleDetail[0]
		pricingPolicy.Price = strconv.FormatFloat(resultModuleDetailPrice.Price, 'f', -1, 64)
	} else {
		return irs.PricingPolicies{}, "", fmt.Errorf("No Price Data")
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

	return pricingPolicy, priceResponse, nil
}

// Util end
