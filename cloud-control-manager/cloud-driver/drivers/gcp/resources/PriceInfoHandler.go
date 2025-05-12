package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"google.golang.org/api/cloudbilling/v1"
	cbb "google.golang.org/api/cloudbilling/v1beta"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

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

	refelectValue = reflect.ValueOf(irs.OnDemand{})

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

			machineTypeMap := make(map[string]*compute.MachineType)

			for _, zone := range zoneList.Items {
				if zoneName, ok := filter["zoneName"]; ok && zone.Name != *zoneName {
					continue
				}

				keepFetching := true
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

					for _, mt := range machineTypes.Items {
						if _, exists := machineTypeMap[mt.Name]; !exists {
							machineTypeMap[mt.Name] = mt
						}
					}
				}
			}

			machineTypeSlice := make([]*compute.MachineType, 0, len(machineTypeMap))
			for _, mt := range machineTypeMap {
				machineTypeSlice = append(machineTypeSlice, mt)
			}

			sort.Slice(machineTypeSlice, func(i, j int) bool {
				return machineTypeSlice[i].Name < machineTypeSlice[j].Name
			})

			if len(machineTypeSlice) > 0 {
				cblogger.Infof("%d machine types have been retrieved", len(machineTypeSlice))

				for _, machineType := range machineTypeSlice {

					if machineTypeFilter, ok := filter["instanceType"]; ok && machineType.Name != *machineTypeFilter {
						continue
					}

					if machineType != nil {
						productInfo, err := mappingToProductInfoForComputePrice(regionName, machineType)

						if err != nil {
							cblogger.Error("error occurred while mapping the product info struct; machine type:", machineType.Name, ", message:", err)
							return "", err
						}

						if productInfoFilter(productInfo, filter) {
							continue
						}

						estimatedCostResponse := &cbb.EstimateCostScenarioForBillingAccountResponse{}

						estimatedCostResponse, err = callEstimateCostScenario(priceInfoHandler, regionName, billingAccountId, machineType)
						if err != nil {
							if googleApiError, ok := err.(*googleapi.Error); ok {
								if googleApiError.Code == 403 {
									return "", errors.New("you don't have any permission to access billing account")
								}
							}

							continue
						}

						priceInfo, err := mappingToPriceInfoForComputePrice(estimatedCostResponse, filter)

						if err != nil {
							cblogger.Error("error occurred while mapping the pricing info struct;; machine type:", machineType.Name, ", message:", err)
							return "", err
						}

						cblogger.Infof("fetch :: %s machine type", productInfo.VMSpecInfo.Name)

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

	cloudPrice := irs.CloudPrice{
		Meta:       irs.Meta{Version: "0.5", Description: "GCP Virtual Machines Price Info"},
		CloudName:  "GCP",
		RegionName: regionName,
		ZoneName:   "NA",
		PriceList:  priceLists,
	}

	convertedPriceData, err := ConvertJsonStringNoEscape(cloudPrice)

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

func callEstimateCostScenario(priceInfoHandler *GCPPriceInfoHandler, region, billingAccountId string, machineType *compute.MachineType) (*cbb.EstimateCostScenarioForBillingAccountResponse, error) {
	machineTypeName := machineType.Name

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
		return nil, err
	}

	return estimateCostScenarioResponse, nil
}

func parseMbToGb(memoryMb int64) float64 {
	return float64(memoryMb) / float64(1<<10)
}

func roundToNearestMultiple(originValue float64) float64 {
	multiple := 0.25

	rounded := math.Round(originValue / multiple)

	return rounded * multiple
}

func (priceInfoHandler *GCPPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	returnProductFamilyNames := []string{}

	returnProductFamilyNames = append(returnProductFamilyNames, "Compute")

	return returnProductFamilyNames, nil
}

func mappingToProductInfoForComputePrice(region string, machineType *compute.MachineType) (*irs.ProductInfo, error) {
	productId := fmt.Sprintf("%d", machineType.Id)

	productInfo := &irs.ProductInfo{
		ProductId:      productId,
		CSPProductInfo: machineType,
	}

	productInfo.VMSpecInfo.Name = machineType.Name
	productInfo.VMSpecInfo.VCpu.Count = fmt.Sprintf("%d", machineType.GuestCpus)
	productInfo.VMSpecInfo.VCpu.ClockGHz = "-1"
	productInfo.VMSpecInfo.MemSizeMiB = fmt.Sprintf("%d", machineType.MemoryMb)
	productInfo.VMSpecInfo.DiskSizeGB = "-1"

	productInfo.Description = machineType.Description

	return productInfo, nil
}

func mappingToPriceInfoForComputePrice(res *cbb.EstimateCostScenarioForBillingAccountResponse, filter map[string]*string) (*irs.PriceInfo, error) {

	result := res.CostEstimationResult
	var onDemand irs.OnDemand
	var cspInfo interface{}

	if len(result.SegmentCostEstimates) > 0 {
		segmentCostEstimate := result.SegmentCostEstimates[0]

		if segmentCostEstimate.SegmentTotalCostEstimate != nil {
			firstWorkloadCostEstimate := segmentCostEstimate.WorkloadCostEstimates[0]

			if firstWorkloadCostEstimate != nil {
				price := firstWorkloadCostEstimate.WorkloadTotalCostEstimate.PreCreditCostEstimate
				parsedPrice := fmt.Sprintf("%d.%09d", price.Units, price.Nanos)
				description := *getDescription(result.Skus, "OnDemand")

				onDemand = irs.OnDemand{
					PricingId:   "NA",
					Unit:        "Hour",
					Currency:    price.CurrencyCode,
					Price:       parsedPrice,
					Description: description,
				}

				cspInfo = firstWorkloadCostEstimate
			}
		}
	}

	return &irs.PriceInfo{
		OnDemand:     onDemand,
		CSPPriceInfo: cspInfo,
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

func getMachineTypeFromSelfLink(selfLink string) string {
	lastSlashIndex := strings.LastIndex(selfLink, "/")

	if lastSlashIndex == -1 {
		return ""
	}

	return selfLink[lastSlashIndex+1:]
}

func getMachineSeriesFromMachineType(machineType string) string {
	firstDashIndex := strings.Index(machineType, "-")

	if firstDashIndex == -1 {
		return ""
	}

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
