package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
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
var validFilterKey map[string]bool

func init() {
	validFilterKey = make(map[string]bool, 0)

	refelectValue := reflect.ValueOf(irs.ProductInfo{})

	for i := 0; i < refelectValue.NumField(); i++ {

		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		if _, ok := validFilterKey[camelCaseFieldName]; !ok {
			validFilterKey[camelCaseFieldName] = true
		}
	}

	refelectValue = reflect.ValueOf(irs.PricingPolicies{})

	for i := 0; i < refelectValue.NumField(); i++ {

		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		if _, ok := validFilterKey[camelCaseFieldName]; !ok {
			validFilterKey[camelCaseFieldName] = true
		}
	}

	refelectValue = reflect.ValueOf(irs.PricingPolicyInfo{})

	for i := 0; i < refelectValue.NumField(); i++ {

		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		if _, ok := validFilterKey[camelCaseFieldName]; !ok {
			validFilterKey[camelCaseFieldName] = true
		}
	}

}

type GCPPriceInfoHandler struct {
	Region               idrv.RegionInfo
	Ctx                  context.Context
	Client               *compute.Service
	BillingCatalogClient *cloudbilling.APIService
	CostEstimationClient *cbb.Service
	Credential           idrv.CredentialInfo
}

// Return the price information of products belonging to the specified Region's PriceFamily in JSON format
func (priceInfoHandler *GCPPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, additionalFilterList []irs.KeyValue) (string, error) {
	priceLists := make([]irs.Price, 0)

	filter, isValid := filterListToMap(additionalFilterList)

	cblogger.Infof(">>> filter key is %v\n", isValid)
	if isValid {
		formattedProjectId := fmt.Sprintf("projects/%s", priceInfoHandler.Credential.ProjectID)
		billingInfo, err := priceInfoHandler.BillingCatalogClient.Projects.GetBillingInfo(formattedProjectId).Do()

		if err != nil {
			cblogger.Error("error while getting billing info for billing account id")
			return "", errors.New("error while getting billing info for billing account id")
		}

		cblogger.Infof("filter value : %+v", additionalFilterList)

		billingAccountId := billingInfo.BillingAccountName

		if billingAccountId == "" || billingAccountId == "billingAccounts/" || !strings.HasPrefix(billingAccountId, "billingAccounts/") {
			cblogger.Error("billing account does not exist on project. connect billing account to current project")
			return "", errors.New("billing account does not exist on project. connect billing account to current project")
		}
		projectID := priceInfoHandler.Credential.ProjectID

		if filteredRegionName, ok := filter["regionName"]; ok {
			regionName = *filteredRegionName
		} else if regionName == "" {
			regionName = priceInfoHandler.Region.Region
		}

		if strings.EqualFold(productFamily, "Compute") {
			regionSelfLink := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s", projectID, regionName)

			zoneList, err := GetZoneListByRegion(priceInfoHandler.Client, projectID, regionSelfLink)
			if err != nil {
				cblogger.Error("error occurred while querying the zone list; ", err)
				return "", err
			}

			machineTypeSlice := make([]*compute.MachineType, 0)

			for _, zone := range zoneList.Items {
				if zoneName, ok := filter["zoneName"]; ok && zone.Name != *zoneName {
					continue
				}

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


					if machineTypeFilter, ok := filter["instanceType"]; ok && machineType.Name != *machineTypeFilter {
						continue
					}

					if machineType != nil {
						// mapping to product info struct
						productInfo, err := mappingToProductInfoForComputePrice(regionName, machineType)

						if err != nil {
							cblogger.Error("error occurred while mapping the product info struct; machine type:", machineType.Name, ", message:", err)
							return "", err
						}

						if productInfoFilter(productInfo, filter) {
							continue
						}

						// call cost estimation api
						estimatedCostResponse, err := callEstimateCostScenario(priceInfoHandler, regionName, billingAccountId, machineType)
						if err != nil {
							cblogger.Error("error occurred when calling the EstimateCostScenario; message:", err)
							
							if googleApiError, ok := err.(*googleapi.Error); ok {
								if googleApiError.Code == 403  {
									return "", errors.New("you don't have any permission to access billing account")
								}
							}

							continue
						}


						// mapping to price info struct
						priceInfo, err := mappingToPriceInfoForComputePrice(estimatedCostResponse, filter)

						if err != nil {
							cblogger.Error("error occurred while mapping the pricing info struct;; machine type:", machineType.Name, ", message:", err)
							return "", err
						}

						cblogger.Infof("fetch :: %s machine type", productInfo.InstanceType)

						priceList := irs.Price{
							ProductInfo: *productInfo,
							PriceInfo:   *priceInfo,
						}


						priceLists = append(priceLists, priceList)
					}
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

func toCamelCase(val string) string {
	if val == "" {
		return ""
	}

	return fmt.Sprintf("%s%s", strings.ToLower(val[:1]), val[1:])
}

func pricePolicyInfoFilter(policy interface{}, filter map[string]*string) bool {
	if len(filter) == 0 {
		return false
	}

	refelectValue := reflect.ValueOf(policy)

	for i := 0; i < refelectValue.NumField(); i++ {

		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		fieldValue := refelectValue.Field(i)

		if invalidRefelctCheck(fieldValue) ||
			fieldValue.Kind() == reflect.Ptr ||
			fieldValue.Kind() == reflect.Struct {
			continue
		}

		fieldStringValue := fmt.Sprintf("%v", fieldValue)

		if value, ok := filter[camelCaseFieldName]; ok {
			skipFlag := value != nil && *value != fieldStringValue
			if skipFlag {
				return true
			}
		}
	}

	return false
}

func priceInfoFilter(policy irs.PricingPolicies, filter map[string]*string) bool {
	if len(filter) == 0 {
		return false
	}

	refelectValue := reflect.ValueOf(policy)

	for i := 0; i < refelectValue.NumField(); i++ {
		fieldName := refelectValue.Type().Field(i).Name
		camelCaseFieldName := toCamelCase(fieldName)
		fieldValue := refelectValue.Field(i)

		if invalidRefelctCheck(fieldValue) ||
			fieldValue.Kind() == reflect.Struct {
			continue
		} else if fieldValue.Kind() == reflect.Ptr {

			derefernceValue := fieldValue.Elem()

			if derefernceValue.Kind() == reflect.Invalid {
				skipFlag := pricePolicyInfoFilter(irs.PricingPolicyInfo{}, filter)
				if skipFlag {
					return true
				}
			} else if derefernceValue.Kind() == reflect.Struct {
				if derefernceValue.Type().Name() == "PricingPolicyInfo" {
					skipFlag := pricePolicyInfoFilter(*policy.PricingPolicyInfo, filter)
					if skipFlag {
						return true
					}
				}
			}
		}

		fieldStringValue := fmt.Sprintf("%v", fieldValue)

		if value, ok := filter[camelCaseFieldName]; ok {
			skipFlag := value != nil && *value != fieldStringValue
			if skipFlag {
				return true
			}
		}
	}

	return false
}

func invalidRefelctCheck(value reflect.Value) bool {
	return value.Kind() == reflect.Array ||
		value.Kind() == reflect.Slice ||
		value.Kind() == reflect.Map ||
		value.Kind() == reflect.Func ||
		value.Kind() == reflect.Interface ||
		value.Kind() == reflect.UnsafePointer ||
		value.Kind() == reflect.Chan
}

func productInfoFilter(productInfo *irs.ProductInfo, filter map[string]*string) bool {
	if len(filter) == 0 {
		return false
	}

	refelectValue := reflect.ValueOf(*productInfo)

	for i := 0; i < refelectValue.NumField(); i++ {
		fieldName := refelectValue.Type().Field(i).Name

		if fieldName == "CSPProductInfo" || fieldName == "Description" {
			continue
		}

		camelCaseFieldName := toCamelCase(fieldName)
		fieldValue := refelectValue.Field(i)

		if invalidRefelctCheck(fieldValue) ||
			fieldValue.Kind() == reflect.Ptr ||
			fieldValue.Kind() == reflect.Struct {
			continue
		}

		fieldStringValue := fmt.Sprintf("%v", fieldValue)

		if value, ok := filter[camelCaseFieldName]; ok {
			skipFlag := value != nil && *value != fieldStringValue

			if skipFlag {
				return true
			}
		}
	}

	return false
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
	productId := fmt.Sprintf("%d", machineType.Id)

	productInfo := &irs.ProductInfo{
		ProductId:      productId,
		RegionName:     region,
		ZoneName:       machineType.Zone,
		CSPProductInfo: machineType,
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

func mappingToPriceInfoForComputePrice(res *cbb.EstimateCostScenarioForBillingAccountResponse, filter map[string]*string) (*irs.PriceInfo, error) {

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

				if !priceInfoFilter(policy, filter) {
					policies = append(policies, policy)
					cspInfo = append(cspInfo, firstWorkloadCostEstimate)
				}
			}
		}

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

					if !priceInfoFilter(policy, filter) {
						policies = append(policies, policy)
						cspInfo = append(cspInfo, commitment)
					}
				}
			}
		}
	}

	return &irs.PriceInfo{
		PricingPolicies: policies,
		CSPPriceInfo:    cspInfo,
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

func filterListToMap(additionalFilterList []irs.KeyValue) (map[string]*string, bool) {
	filterMap := make(map[string]*string, 0)

	if additionalFilterList == nil {
		return filterMap, true
	}

	for _, kv := range additionalFilterList {
		if _, ok := validFilterKey[kv.Key]; !ok {
			return map[string]*string{}, false
		}

		value := strings.TrimSpace(kv.Value)
		if value == "" {
			continue
		}

		filterMap[kv.Key] = &value
	}

	return filterMap, true
}
