package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"google.golang.org/api/cloudbilling/v1"
	cbb "google.golang.org/api/cloudbilling/v1beta"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// API를 호출하는 데 특정 IAM 권한이 필요하지 않습니다.
// https://cloudbilling.googleapis.com/v2beta/services?key=API_KEY&pageSize=PAGE_SIZE&pageToken=PAGE_TOKEN

// sku
// https://cloud.google.com/skus/?currency=USD&filter=38FA-6071-3D88&hl=ko
const ()

type GCPPriceInfoHandler struct {
	Region               idrv.RegionInfo
	Ctx                  context.Context
	Client               *compute.Service
	BillingCatalogClient *cloudbilling.APIService
	CostEstimationClient *cbb.Service
	Credential           idrv.CredentialInfo
}

// Return the price information of products belonging to the specified Region's PriceFamily in JSON format
func (priceInfoHandler *GCPPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter []irs.KeyValue) (string, error) {

	billindAccountId := priceInfoHandler.Credential.BillingAccountID

	if billindAccountId == "" || billindAccountId == "billingAccounts/" {
		cblogger.Error("billing accout id does not exist")
		return "", errors.New("billing account is a mandatory field")
	}

	if regionName == "" {
		regionName = priceInfoHandler.Region.Region
	}

	projectID := priceInfoHandler.Credential.ProjectID
	priceLists := make([]irs.Price, 0)

	if strings.EqualFold(productFamily, "Compute") {
		regionSelfLink := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s", projectID, regionName)

		zoneList, err := GetZoneListByRegion(priceInfoHandler.Client, projectID, regionSelfLink)
		if err != nil {
			cblogger.Error("error occurred while querying the zone list; ", err)
			return "", err
		}

		/*
		 * compute.MachineType 에서 머신 타입이 동일하면 zone 정보를 제외하고 모두 동일 정보 제공
		 * zone 정보만 다르게 해서 동일한 machine type 으로 호출에 대한 최적화 가능
		 */
		machineTypeSlice := make([]*compute.MachineType, 0)

		for _, zone := range zoneList.Items {

			keepFetching := true // machine type 조회 반복 호출 flag
			nextPageToken := ""

			for keepFetching {

				machineTypes, err := priceInfoHandler.Client.MachineTypes.List(projectID, zone.Name).Do(googleapi.QueryParameter("pageToken", nextPageToken))

				if err != nil {
					cblogger.Error("error occurred while querying the machine type list; zone:", zone.Name, ", message:", err)
					return "", err
				}

				if keepFetching = machineTypes.NextPageToken != ""; keepFetching {
					nextPageToken = machineTypes.NextPageToken
				}

				machineTypeSlice = append(machineTypeSlice, machineTypes.Items...)
			}
		}

		if len(machineTypeSlice) > 0 {
			cblogger.Infof("%d machine types have been retrieved", len(machineTypeSlice))

			for _, machineType := range machineTypeSlice {

				if machineType != nil {
					// mapping to product info struct
					productInfo, err := mappingToProductInfoForComputePrice(regionName, machineType)

					if err != nil {
						cblogger.Error("error occurred while mapping the product info struct; machine type:", machineType.Name, ", message:", err)
						return "", err
					}

					// call cost estimation api
					estimatedCostResponse, err := callEstimateCostScenario(priceInfoHandler, regionName, billindAccountId, machineType)
					if err != nil {
						cblogger.Error("error occurred when calling the EstimateCostScenario; message:", err)
						continue
					}

					// mapping to price info struct
					priceInfo, err := mappingToPriceInfoForComputePrice(estimatedCostResponse)

					if err != nil {
						cblogger.Error("error occurred while mapping the pricing info struct;; machine type:", machineType.Name, ", message:", err)
						return "", err
					}

					priceList := irs.Price{
						ProductInfo: *productInfo,
						PriceInfo:   *priceInfo,
					}

					priceLists = append(priceLists, priceList)
				}

			}
		}
	}

	cloudPriceData := irs.CloudPriceData{
		Meta: irs.Meta{
			Version:     "v0.1",
			Description: "Multi-Cloud Price Info Api",
		},
		CloudPriceList: []irs.CloudPrice{
			{
				CloudName: "GCP",
				PriceList: priceLists,
			},
		},
	}

	convertedPriceData, err := ConvertJsonStringNoEscape(cloudPriceData)

	if err != nil {
		cblogger.Error("error occurred when removing escape characters from the response struct;", err)
		return "", err
	}

	return convertedPriceData, nil

}

/* @Info
 * billingAccountId format - billingAccounts/xxxx-xxxx-xxxx
 */
func callEstimateCostScenario(priceInfoHandler *GCPPriceInfoHandler, region, billingAccountId string, machineType *compute.MachineType) (*cbb.EstimateCostScenarioForBillingAccountResponse, error) {
	machineTypeName := getMachineTypeFromSelfLink(machineType.SelfLink)
	if machineTypeName == "" {
		return nil, errors.New("machine type is not defined")
	}

	machineSeries := getMachineSeriesFromMachineType(machineTypeName)
	if machineSeries == "" {
		return nil, errors.New("machine series is not defined")
	}

	vCpu := machineType.GuestCpus
	memory := roundToNearestMultiple(parseMbToGb(machineType.MemoryMb))

	estimateCostScenarioResponse, err := priceInfoHandler.CostEstimationClient.BillingAccounts.EstimateCostScenario(
		billingAccountId,
		&cbb.EstimateCostScenarioForBillingAccountRequest{
			CostScenario: &cbb.CostScenario{
				Workloads: []*cbb.Workload{
					{
						ComputeVmWorkload: &cbb.ComputeVmWorkload{
							MachineType: &cbb.MachineType{
								PredefinedMachineType: &cbb.PredefinedMachineType{
									MachineType: machineTypeName,
								},
							},
							Region: region,
							InstancesRunning: &cbb.Usage{
								UsageRateTimeline: &cbb.UsageRateTimeline{
									UsageRateTimelineEntries: []*cbb.UsageRateTimelineEntry{
										{
											UsageRate: 1,
										},
									},
								},
							},
						},
						Name: "ondemand-instance-workload-price",
					},
				},
				ScenarioConfig: &cbb.ScenarioConfig{
					EstimateDuration: "3600s",
				},
				Commitments: []*cbb.Commitment{
					{
						Name: "1yrs-commitment-price",
						VmResourceBasedCud: &cbb.VmResourceBasedCud{
							Region:          region,
							VirtualCpuCount: vCpu,
							MemorySizeGb:    memory,
							Plan:            "TWELVE_MONTH",
							MachineSeries:   machineSeries,
						},
					},
					{
						Name: "3yrs-commitment-price",
						VmResourceBasedCud: &cbb.VmResourceBasedCud{
							Region:          region,
							VirtualCpuCount: vCpu,
							MemorySizeGb:    memory,
							Plan:            "THIRTY_SIX_MONTH",
							MachineSeries:   machineSeries,
						},
					},
				},
			},
		},
	).Do()

	if err != nil {
		cblogger.Errorf("error occurred when calling EstimateCostScenario; machine spec; machine type: %s, memory: %d, calculated memory: %f", machineType.Name, machineType.MemoryMb, memory)
		return nil, err
	}

	return estimateCostScenarioResponse, nil
}

/*
 * parse mb to gb
 * mb memory devide to 2^10 = 1024
 */
func parseMbToGb(memoryMb int64) float64 {
	return float64(memoryMb) / float64(1<<10)
}

/*
 * obtain the closest multiple of 0.25 to the origin value.
 */
func roundToNearestMultiple(originValue float64) float64 {
	multiple := 0.25

	// Round the result of dividing "value" by "multiple" to the nearest integer
	rounded := math.Round(originValue / multiple)

	// multiply "rounded" by "multiple," you will get the closest multiple of the original value.
	return rounded * multiple
}

/*
 * BillingCatalogClient.Services.Skus.List()을 호출하여 가져온 Category.ResourceFamily 를 중복 제거하여 리스트 생성
 */
func (priceInfoHandler *GCPPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	returnProductFamilyNames := []string{}

	returnProductFamilyNames = append(returnProductFamilyNames, "Compute")
	// returnProductFamilyNames = append(returnProductFamilyNames, "License")
	// returnProductFamilyNames = append(returnProductFamilyNames, "Network")
	// returnProductFamilyNames = append(returnProductFamilyNames, "Search")
	// returnProductFamilyNames = append(returnProductFamilyNames, "Storage")
	// returnProductFamilyNames = append(returnProductFamilyNames, "Utility")

	return returnProductFamilyNames, nil
}

func mappingToProductInfoForComputePrice(region string, machineType *compute.MachineType) (*irs.ProductInfo, error) {
	cspProductInfoString, err := json.Marshal(*machineType)

	if err != nil {
		return &irs.ProductInfo{}, err
	}

	productId := fmt.Sprintf("%d", machineType.Id)

	productInfo := &irs.ProductInfo{
		ProductId:      productId,
		RegionName:     region,
		ZoneName:       machineType.Zone,
		CSPProductInfo: string(cspProductInfoString),
	}

	productInfo.InstanceType = machineType.Name
	productInfo.Vcpu = fmt.Sprintf("%d", machineType.GuestCpus)
	productInfo.Memory = fmt.Sprintf("%.2f GB", roundToNearestMultiple(parseMbToGb(machineType.MemoryMb)))
	productInfo.Description = machineType.Description

	productInfo.Gpu = "NA"
	productInfo.Storage = "NA"
	productInfo.GpuMemory = "NA"
	productInfo.OperatingSystem = "NA"
	productInfo.PreInstalledSw = "NA"

	productInfo.VolumeType = ""
	productInfo.StorageMedia = ""
	productInfo.MaxVolumeSize = ""
	productInfo.MaxIOPSVolume = ""
	productInfo.MaxThroughputVolume = ""

	return productInfo, nil
}

/*
	@GCP 가격 정책
	ListPrice => list price -> 정가 (cpu + ram)
	ContractPrice => contract price -> 계약 가격 (cpu + ram + storage, disk 등)
	CUD => committed use discount (CUD) -> 약정 각격 (cpu + ram + 약정 + a(storage, disk 등))
		1YearCUD
		3YearCUD
*/

func mappingToPriceInfoForComputePrice(res *cbb.EstimateCostScenarioForBillingAccountResponse) (*irs.PriceInfo, error) {

	result := res.CostEstimationResult
	policies := make([]irs.PricingPolicies, 0)
	cspInfo := make([]interface{}, 0)

	if len(result.SegmentCostEstimates) > 0 {
		segmentCostEstimate := result.SegmentCostEstimates[0]

		// mapping from GCP OnDemand price struct to PricingPolicies struct
		if segmentCostEstimate.SegmentTotalCostEstimate != nil {
			firstWorkloadCostEstimate := segmentCostEstimate.WorkloadCostEstimates[0]

			if firstWorkloadCostEstimate != nil {
				price := firstWorkloadCostEstimate.WorkloadTotalCostEstimate.PreCreditCostEstimate
				parsedPrice := fmt.Sprintf("%d.%09d", price.Units, price.Nanos)
				description := *getDescription(result.Skus, "OnDemand")

				policy := irs.PricingPolicies{
					PricingId:     "NA",
					PricingPolicy: "OnDemand",
					Unit:          "Hrs",
					Currency:      price.CurrencyCode,
					Price:         parsedPrice,
					Description:   description,
				}

				policies = append(policies, policy)
				cspInfo = append(cspInfo, segmentCostEstimate.SegmentTotalCostEstimate)
			}

		}

		// mapping from GCP Commitment price struct to PricingPolicies struct
		if len(segmentCostEstimate.CommitmentCostEstimates) > 0 {
			for _, commitment := range segmentCostEstimate.CommitmentCostEstimates {
				if commitment.CommitmentTotalCostEstimate != nil {
					priceStruct := commitment.CommitmentTotalCostEstimate.NetCostEstimate

					pricingPolicy := "Commit1Yr"
					contract := "1yr"

					if commitment.Name == "3yrs-commitment-price" {
						pricingPolicy = "Commit3Yr"
						contract = "3yr"
					}

					pricingPolicyInfo := &irs.PricingPolicyInfo{
						LeaseContractLength: contract,
						OfferingClass:       "NA",
						PurchaseOption:      "NA",
					}

					description := *getDescription(result.Skus, "Commitment")

					policy := irs.PricingPolicies{
						PricingId:         "NA",
						PricingPolicy:     pricingPolicy,
						Unit:              "Yrs",
						Currency:          priceStruct.CurrencyCode,
						Price:             fmt.Sprintf("%d.%09d", priceStruct.Units, priceStruct.Nanos),
						Description:       description,
						PricingPolicyInfo: pricingPolicyInfo,
					}

					policies = append(policies, policy)
					cspInfo = append(cspInfo, commitment.CommitmentTotalCostEstimate)
				}
			}
		}
	}

	marshalledCspInfo, err := json.Marshal(cspInfo)

	if err != nil {
		cblogger.Error("error occurred during the marshaling process of cspinfo; ", err)
		marshalledCspInfo = []byte("")
	}

	return &irs.PriceInfo{
		PricingPolicies: policies,
		CSPPriceInfo:    string(marshalledCspInfo),
	}, nil
}

func getDescription(skus []*cbb.Sku, condition string) *string {
	description := ""

	if len(skus) > 0 {
		for _, sku := range skus {
			if condition == "Commitment" {
				if strings.HasPrefix(sku.DisplayName, "Commitment") {
					if len(description) == 0 {
						description = sku.DisplayName
					} else {
						description = fmt.Sprintf("%s / %s", description, sku.DisplayName)
					}
				}
			} else if condition == "OnDemand" {
				if !strings.HasPrefix(sku.DisplayName, "Commitment") {
					if len(description) == 0 {
						description = sku.DisplayName
					} else {
						description = fmt.Sprintf("%s / %s", description, sku.DisplayName)
					}
				}
			}
		}
	}

	return &description
}

// Convert from Cloud Object to JSON String type
func ConvertJsonStringNoEscape(v interface{}) (string, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	errJson := encoder.Encode(v)

	if errJson != nil {
		cblogger.Error("fail to convert json string", errJson)
		return "", errJson
	}

	jsonString := buffer.String()
	jsonString = strings.Replace(jsonString, "\\", "", -1)

	return jsonString, nil
}

// Extracting machine type through the self-link
func getMachineTypeFromSelfLink(selfLink string) string {
	// Finding the index of the last '/' character
	lastSlashIndex := strings.LastIndex(selfLink, "/")

	if lastSlashIndex == -1 {
		return ""
	}

	// Extracting the substring after the last '/'
	return selfLink[lastSlashIndex+1:]
}

// machine type 을 통해서 machine series 추출
func getMachineSeriesFromMachineType(machineType string) string {
	// 마지막 / 의 인덱스 찾기
	firstDashIndex := strings.Index(machineType, "-")

	if firstDashIndex == -1 {
		return ""
	}

	// 마지막 / 뒤의 부분 문자열 추출
	return machineType[:firstDashIndex]
}

/*********************************************************************/
/*************************** Code Archive ****************************/
/*********************************************************************/
// 실제 billing service를 호출하여 결과 확인
func CallServicesList(priceInfoHandler *GCPPriceInfoHandler) ([]string, error) {
	returnProductFamilyNames := []string{}
	// ***STEP1 : Services.List를 호출하여 모든 Servic를 조회 : 상세조건에 해당하는 api가 현재 없음.*** ///

	//resp, err := priceInfoHandler.CloudBillingClient.Services.Skus.List("services/0017-8C5E-5B91").Do()
	// priceInfoHandler.CloudBillingApiClient.Services.List().Fields("services") // 해당 결과에서 원하는 Field만 조회할 때 사용 ex) services.name : services > name 만 가져온다. 여러 건의 경우 콤마로 구분 services.name,services.displayName
	respService, err := priceInfoHandler.BillingCatalogClient.Services.List().Do() // default 5000건.
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
		resp, err := priceInfoHandler.BillingCatalogClient.Services.Skus.List(serviceName).Do()

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

		resp, err := priceInfoHandler.BillingCatalogClient.Services.Skus.List(parent).PageToken(nextPageToken).Do()

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
	priceList := []irs.Price{}
	for _, sku := range skuArr {
		aPrice := irs.Price{}
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
