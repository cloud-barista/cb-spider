package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"google.golang.org/api/cloudbilling/v1"
	compute "google.golang.org/api/compute/v1"
)

// API를 호출하는 데 특정 IAM 권한이 필요하지 않습니다.
// https://cloudbilling.googleapis.com/v2beta/services?key=API_KEY&pageSize=PAGE_SIZE&pageToken=PAGE_TOKEN

// sku
// https://cloud.google.com/skus/?currency=USD&filter=38FA-6071-3D88&hl=ko
const DEFAULT_PAGE_SIZE = 5000

type GCPPriceInfoHandler struct {
	Region                idrv.RegionInfo
	Ctx                   context.Context
	Client                *compute.Service
	CloudBillingApiClient *cloudbilling.APIService

	//Client     *billing.CloudBillingClient
	Credential idrv.CredentialInfo
}

// 해당 Region의 PriceFamily에 해당하는 제품들의 가격정보를 json형태로 return
func (priceInfoHandler *GCPPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter []irs.KeyValue) (string, error) {
	returnJson := ""

	// Compute Engine SKU 및 가격 정보 가져오기

	// VM의 경우 아래 항목에 대해 가격이 매겨짐.
	// VM 인스턴스 가격 책정
	// 네트워킹 가격 책정
	// 단독 테넌트 노드 가격 책정
	// GPU 가격 책정
	// 디스크 및 이미지 가격 책정
	serviceID := ""
	switch productFamily {
	case "ApplicationServices":
		serviceID = ""
	case "Compute": // Service Description : Compute Engine
		serviceID = "6F81-5844-456A"
	case "License":
		serviceID = ""
	case "Network": // Service Description : Networking
		serviceID = "E505-1604-58F8"
	case "Search": // Service Description : Elastic Cloud (Elasticsearch managed service)
		serviceID = "6F81-5844-456A"
	case "Storage": // Service Description : Cloud Storage
		serviceID = "95FF-2EF5-5EA1"
	case "Utility":
		serviceID = ""
	default:
		serviceID = ""
	}

	if serviceID == "" {
		return "", errors.New("Unsupported productFamily. " + productFamily)
	}

	parent := "services/" + serviceID
	listSkus, err := CallServicesSkusList(priceInfoHandler, parent)
	if err != nil {

	}
	log.Println(listSkus)

	// projectID := priceInfoHandler.Credential.ProjectID
	// resp, err := GetRegion(priceInfoHandler.Client, projectID, regionName)
	// if err != nil {
	// 	cblogger.Error(err)
	// 	return returnJson, err
	// }
	// cblogger.Debug(resp)

	return returnJson, nil
}

// product family의 이름들을 배열로 return
// CallServicesList()을 호출하여 가져온 Category.ResourceFamily를 하드코딩
func (priceInfoHandler *GCPPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	returnProductFamilyNames := []string{}

	returnProductFamilyNames = append(returnProductFamilyNames, "ApplicationServices")
	returnProductFamilyNames = append(returnProductFamilyNames, "Compute")
	returnProductFamilyNames = append(returnProductFamilyNames, "License")
	returnProductFamilyNames = append(returnProductFamilyNames, "Network")
	returnProductFamilyNames = append(returnProductFamilyNames, "Search")
	returnProductFamilyNames = append(returnProductFamilyNames, "Storage")
	returnProductFamilyNames = append(returnProductFamilyNames, "Utility")

	return returnProductFamilyNames, nil
}

// 실제 billing service를 호출하여 결과 확인
func CallServicesList(priceInfoHandler *GCPPriceInfoHandler) ([]string, error) {
	returnProductFamilyNames := []string{}
	// ***STEP1 : Services.List를 호출하여 모든 Servic를 조회 : 상세조건에 해당하는 api가 현재 없음.*** ///

	//resp, err := priceInfoHandler.CloudBillingClient.Services.Skus.List("services/0017-8C5E-5B91").Do()
	// priceInfoHandler.CloudBillingApiClient.Services.List().Fields("services") // 해당 결과에서 원하는 Field만 조회할 때 사용 ex) services.name : services > name 만 가져온다. 여러 건의 경우 콤마로 구분 services.name,services.displayName
	respService, err := priceInfoHandler.CloudBillingApiClient.Services.List().Do() // default 5000건.
	//respService, err := priceInfoHandler.CloudBillingApiClient.Services.List().PageSize(10).PageToken("").Do() // 만약 total count 가 5000 이상이면 pageSize와 pageToken을 이용해 조회 필요. 다음페이지가 없으면 nextPageToken은 "" 임
	// ///////// 가져오는 결과 형태 /////////////
	// (*cloudbilling.Service)(0xc0002ddc70)({
	// 	BusinessEntityName: (string) (len=20) "businessEntities/GCP",
	// 	DisplayName: (string) (len=24) "ADFS Windows Server 2016",
	// 	Name: (string) (len=23) "services/EEF5-99AE-6778",
	// 	ServiceId: (string) (len=14) "EEF5-99AE-6778",
	// 	ForceSendFields: ([]string) <nil>,
	// 	NullFields: ([]string) <nil>
	//    }),
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	// // spew.Dump(respService)
	// for _, service := range respService.Services {
	// 	category := service.
	// }

	categoryResourceFamily := map[string]string{}
	categoryResourceGroup := map[string]string{}
	categoryServiceDisplayName := map[string]string{}
	totalCnt := 0

	// ***STEP2 : Services.List에서 service의 name으로 Sku 목록 조회 *** ///
	for _, service := range respService.Services {
		totalCnt++
		//resp, err := priceInfoHandler.CloudBillingApiClient.Services.Skus.List("services/6F81-5844-456A").Do()
		serviceName := service.Name
		resp, err := priceInfoHandler.CloudBillingApiClient.Services.Skus.List(serviceName).Do()

		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
		//spew.Dump(resp)
		i := 0

		// ***STEP3 : Sku에서 Category 안에 있는 ResourceFamily를 map에 담아 중복제거 *** ///
		for _, sku := range resp.Skus {

			if sku.Category.ResourceFamily != "Compute" {
				fmt.Println("ski resourceFamily = ", sku.Category.ResourceFamily)
				continue
			}

			//spew.Dump(sku)
			i++

			categoryResourceFamily[sku.Category.ResourceFamily] = sku.Category.ResourceFamily
			categoryResourceGroup[sku.Category.ResourceGroup] = sku.Category.ResourceGroup
			categoryServiceDisplayName[sku.Category.ServiceDisplayName] = sku.Category.ServiceDisplayName

			// log.Println("sku name ", sku.Name)
			// log.Println("sku id ", sku.SkuId)
			// log.Println("category ", sku.Category)
			// log.Println("serviceRegions ", sku.ServiceRegions)

			// Category: (*cloudbilling.Category)(0xc0004d00e0)({
			// 	ResourceFamily: (string) (len=7) "Compute",
			// 	ResourceGroup: (string) (len=3) "GPU",
			// 	ServiceDisplayName: (string) (len=14) "Compute Engine",
			// 	UsageType: (string) (len=11) "Preemptible",
			// 	ForceSendFields: ([]string) <nil>,
			// 	NullFields: ([]string) <nil>
			//    }),

		} // end of skus
		// log.Println(serviceName, ", i= ", i)
		fmt.Println(serviceName, ", i= ", i)
	} // end of service
	// log.Println(" categoryResourceFamily= ", categoryResourceFamily)
	// log.Println(" categoryResourceGroup= ", categoryResourceGroup)
	// log.Println(" categoryServiceDisplayName= ", categoryServiceDisplayName)
	fmt.Println(" categoryResourceFamily= ", categoryResourceFamily)
	fmt.Println(" categoryResourceGroup= ", categoryResourceGroup)
	fmt.Println(" categoryServiceDisplayName= ", categoryServiceDisplayName)
	fmt.Println(" totalCnt = ", totalCnt)

	// ***STEP4 : ResourceFamily Map을 string array로 변경하여 return *** ///
	for key := range categoryResourceFamily {
		fmt.Printf("Key: %s\n", key)
		returnProductFamilyNames = append(returnProductFamilyNames, key)
	}

	return returnProductFamilyNames, nil
}

// 실제 billing services > skus 를 호출하여 결과 확인
// parent = services/{serviceId}
func CallServicesSkusList(priceInfoHandler *GCPPriceInfoHandler, parent string) (*cloudbilling.ListSkusResponse, error) {
	log.Println(" parent ", parent)

	// nextToken이 없어질 때까지 반복.
	hasNextToken := 1
	nextPageToken := ""
	//skuArr := []*cloudbilling.ListSkusResponse{}
	skuArr := []*cloudbilling.Sku{}
	for hasNextToken > 0 {

		resp, err := priceInfoHandler.CloudBillingApiClient.Services.Skus.List(parent).PageToken(pageToken).Do()

		if err != nil {

		}
		skuArr = append(skuArr, resp.Skus...)

		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			hasNextToken = 0
			break
		}
		log.Println(resp)
	}

	// 가져온 respArr을 mapping 한다.
	cloudPriceData := irs.CloudPriceData{} // 가장 큰 단위( meta 포함 )
	cloudPriceList := []irs.CloudPrice{}   // meta를 제외한 가장 큰 단위
	cloudPrice := irs.CloudPrice{}         // 해당 cloud의 모든 price 정보
	priceList := []irs.PriceList{}
	for _, sku := range skuArr {
		aPrice := irs.PriceList{}
		priceInfo := irs.PriceInfo{}

		// priceInfo.PricingPolicies

		skuPriceInforArr := sku.PricingInfo
		pricePolicies := []irs.PricingPolicies{}
		for _, pricing := range skuPriceInforArr {
			pricePolicy := irs.PricingPolicies{}
			pricePolicy.PricingId = sku.SkuId

			//"usageType": "OnDemand", "Preemptible", "Commit1Yr" ...
			pricePolicy.PricingPolicy = sku.Category.UsageType

			// price는 계산해야 함.
			// baseUnitConversionFactor * (tieredRates.units + tieredRates.nanos)
			mappingPrice(pricePolicy, pricing.PricingExpression)

			// Price             string             `json:"price"`
			// Description       string             `json:"description"`
			// PricingPolicyInfo *PricingPolicyInfo `json:"pricingPolicyInfo,omitempty"`

			pricePolicies = append(pricePolicies, pricePolicy)
		}
		priceInfo.PricingPolicies = pricePolicies

		priceList = append(priceList, aPrice)
	}
	// type PriceList struct {
	// 	ProductInfo ProductInfo `json:"productInfo"`
	// 	PriceInfo   PriceInfo   `json:"priceInfo"`
	// }
	cloudPriceList = append(cloudPriceList, cloudPrice)
	cloudPriceData.CloudPriceList = cloudPriceList

	return nil, nil
	//return resp, err
}

// 가격 계산
// 가격 계산 식:
// 가격=(전체 단위+나노초109)×단위 가격가격=(전체 단위+109나노초​)×단위 가격
//
//	전체 단위전체 단위: units 필드의 값
//	나노초나노초: nanos 필드의 값
//	단위 가격단위 가격: unitPrice의 units와 nanos를 이용하여 구한 1초당 가격
//	가격가격: 최종적으로 계산된 가격
func mappingPrice(pricePolicy irs.PricingPolicies, pricingExpression *cloudbilling.PricingExpression) {

	//func calculatePrice(unitPrice float64, usageSeconds float64, conversionFactor float64) float64 {
	baseUnit := pricingExpression.BaseUnit                                 // 전체단위
	baseUnitConversionFactor := pricingExpression.BaseUnitConversionFactor // 환산에 필요한 값
	usageUnit := pricingExpression.UsageUnit                               // 표시단위 ( h = 3600s )
	tieredRates := pricingExpression.TieredRates

	calPrice := float64(0)

	// TiredRates가 배열이므로 USD 등을 찾아야 함.
	for _, tier := range tieredRates {
		currencyCode := tier.UnitPrice.CurrencyCode
		// if currencyCode != "USD" {
		// 	continue
		// } // USD 만 계산.

		nanos := float64(tier.UnitPrice.Nanos)
		units := float64(tier.UnitPrice.Units)
		if baseUnit != usageUnit {
			calPrice = (units + nanos/1e9) * baseUnitConversionFactor
		} else {
			calPrice = (units + nanos/1e9)
		}
		pricePolicy.Currency = currencyCode
		pricePolicy.Unit = usageUnit
		pricePolicy.Price = strconv.FormatFloat(calPrice, 'f', -1, 64)
		pricePolicy.Description = fmt.Sprintf("units = %s , nanos = %.2f", units, nanos)
	}

	//unitPrice * (usageSeconds / conversionFactor)

	// "usageUnit": "h",
	//         "displayQuantity": 1,
	//         "tieredRates": [
	//           {
	//             "startUsageAmount": 0,
	//             "unitPrice": {
	//               "currencyCode": "USD",
	//               "units": "0",
	//               "nanos": 20550000 ->0.02055
	//             }
	//           }
	//         ],
	//         "usageUnitDescription": "hour",
	//         "baseUnit": "s",
	//         "baseUnitDescription": "second",
	//         "baseUnitConversionFactor": 3600
	//       },
	//       "currencyConversionRate": 1,

	// SKU 비용은 units + nanos입니다. 예를 들어 $1.75 비용은 units=1 및 nanos=750,000,000으로 나타냅니다.
	// 단위 설명
	// 사용량 가격 등급 시작액

}

// unit은 더하고
func calculatePrice(units int64, nanos int, unitPrice float64, baseUnitConversionFactor float64) float64 {
	// baseUnit을 시간으로 변환
	hours := float64(units*int64(baseUnitConversionFactor)) / 3600

	// 가격 계산
	return (hours + float64(nanos)/1e9) * unitPrice
}

/*
Commitment v1: A2 Cpu in APAC for 1 Year
38FA-6071-3D88	0.0230593 USD per 1 hour

{
      "name": "services/6F81-5844-456A/skus/38FA-6071-3D88",
      "skuId": "38FA-6071-3D88",
      "description": "Commitment v1: A2 Cpu in APAC for 1 Year",
      "category": {
        "serviceDisplayName": "Compute Engine",
        "resourceFamily": "Compute",
        "resourceGroup": "CPU",
        "usageType": "Commit1Yr"
      },
      "serviceRegions": [
        "asia-east1"
      ],
      "pricingInfo": [
        {
          "summary": "",
          "pricingExpression": {
            "usageUnit": "h",
            "displayQuantity": 1,
            "tieredRates": [
              {
                "startUsageAmount": 0,
                "unitPrice": {
                  "currencyCode": "USD",
                  "units": "0",
                  "nanos": 23059300
                }
              }
            ],
            "usageUnitDescription": "hour",
            "baseUnit": "s",
            "baseUnitDescription": "second",
            "baseUnitConversionFactor": 3600
          },
          "currencyConversionRate": 1,
          "effectiveTime": "2023-12-20T22:56:00.158911Z"
        }
      ],
      "serviceProviderName": "Google",
      "geoTaxonomy": {
        "type": "REGIONAL",
        "regions": [
          "asia-east1"
        ]
      }
    },

*/
