package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzurePriceInfoHandler struct {
	CredentialInfo     idrv.CredentialInfo
	Region             idrv.RegionInfo
	Ctx                context.Context
	ResourceSkusClient *armcompute.ResourceSKUsClient
}

const AzurePriceApiEndpoint = "https://prices.azure.com/api/retail/prices"

type Item struct {
	CurrencyCode         string  `json:"currencyCode"`
	TierMinimumUnits     float64 `json:"tierMinimumUnits"`
	RetailPrice          float64 `json:"retailPrice"`
	UnitPrice            float64 `json:"unitPrice"`
	ArmRegionName        string  `json:"armRegionName"`
	Location             string  `json:"location"`
	EffectiveStartDate   string  `json:"effectiveStartDate"`
	MeterID              string  `json:"meterId"`
	MeterName            string  `json:"meterName"`
	ProductID            string  `json:"productId"`
	SkuID                string  `json:"skuId"`
	ProductName          string  `json:"productName"`
	SkuName              string  `json:"skuName"`
	ServiceName          string  `json:"serviceName"`
	ServiceID            string  `json:"serviceId"`
	ServiceFamily        string  `json:"serviceFamily"`
	UnitOfMeasure        string  `json:"unitOfMeasure"`
	Type                 string  `json:"type"`
	IsPrimaryMeterRegion bool    `json:"isPrimaryMeterRegion"`
	ArmSkuName           string  `json:"armSkuName"`
	ReservationTerm      string  `json:"reservationTerm,omitempty"`
	EffectiveEndDate     string  `json:"effectiveEndDate,omitempty"`
}

type PriceInfo struct {
	BillingCurrency    string `json:"BillingCurrency"`
	CustomerEntityID   string `json:"CustomerEntityId"`
	CustomerEntityType string `json:"CustomerEntityType"`
	Items              []Item `json:"items"`
	NextPageLink       string `json:"NextPageLink"`
	Count              int    `json:"Count"`
}

func getAzurePriceInfo(filterOption string) ([]byte, error) {
	URL := AzurePriceApiEndpoint + "?$filter=" + url.QueryEscape(filterOption)

	ctx := context.Background()
	client := &http.Client{}

	var jsonResponse map[string]interface{}

	for URL != "" {
		req, err := http.NewRequest(http.MethodGet, URL, nil)
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		responseBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var currentPage map[string]interface{}
		err = json.Unmarshal(responseBody, &currentPage)
		if err != nil {
			return nil, err
		}

		if jsonResponse == nil {
			jsonResponse = currentPage
		} else {
			if items, ok := jsonResponse["Items"].([]interface{}); ok {
				if nextItems, ok := currentPage["Items"].([]interface{}); ok {
					jsonResponse["Items"] = append(items, nextItems...)
				}
			}
		}

		if nextURL, ok := currentPage["NextPageLink"].(string); ok && nextURL != "" {
			URL = nextURL
		} else {
			break
		}
	}

	if jsonResponse != nil {
		return json.Marshal(jsonResponse)
	}

	return nil, fmt.Errorf("no data received")
}

func isFieldToFilterExist(structVal any, filterList []irs.KeyValue) (exist bool, fields []string) {
	var val reflect.Value

	if len(filterList) == 0 {
		return false, fields
	}

	if _, ok := structVal.(irs.ProductInfo); ok {
		data := structVal.(irs.ProductInfo)
		val = reflect.ValueOf(&data).Elem()
	} else if _, ok := structVal.(irs.PricingPolicies); ok {
		data := structVal.(irs.PricingPolicies)
		val = reflect.ValueOf(&data).Elem()
	} else {
		return false, fields
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i).Name
		fields = append(fields, field)
	}

	for _, filter := range filterList {
		for _, field := range fields {
			fieldToLower := strings.ToLower(field)
			keyToLower := strings.ToLower(filter.Key)
			if keyToLower == fieldToLower {
				return true, fields
			}
		}
	}

	return false, fields
}

func isPicked(structVal any, fields []string, filterList []irs.KeyValue) bool {
	var val reflect.Value

	if _, ok := structVal.(irs.ProductInfo); ok {
		data := structVal.(irs.ProductInfo)
		val = reflect.ValueOf(&data).Elem()
	} else if _, ok := structVal.(irs.PricingPolicies); ok {
		data := structVal.(irs.PricingPolicies)
		val = reflect.ValueOf(&data).Elem()
	} else {
		return false
	}

	if len(filterList) == 0 {
		return true
	}

	for _, filter := range filterList {
		for _, field := range fields {
			fieldToLower := strings.ToLower(field)
			keyToLower := strings.ToLower(filter.Key)
			if keyToLower == fieldToLower {
				fieldValue := reflect.Indirect(val).FieldByName(field).String()
				fieldValueToLower := strings.ToLower(fieldValue)
				valueToLower := strings.ToLower(filter.Value)
				if fieldValueToLower == valueToLower {
					return true
				}
			}
		}
	}

	return false
}

func (priceInfoHandler *AzurePriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "ListProductFamily()")
	start := call.Start()

	result, err := getAzurePriceInfo("armRegionName eq '" + regionName + "'")
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	LoggingInfo(hiscallInfo, start)

	var priceInfo PriceInfo
	err = json.Unmarshal(result, &priceInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	var serviceFamilyList []string
	for _, item := range priceInfo.Items {
		serviceFamilyList = append(serviceFamilyList, item.ServiceFamily)
	}
	serviceFamilyList = removeDuplicateStr(serviceFamilyList)

	return serviceFamilyList, nil
}

func (priceInfoHandler *AzurePriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "ListProductFamily()")
	start := call.Start()

	filterOption := "serviceName eq 'Virtual Machines'" + " and priceType eq 'Consumption'" + " and armRegionName eq '" + regionName + "'"

	result, err := getAzurePriceInfo(filterOption)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	LoggingInfo(hiscallInfo, start)

	var priceInfo PriceInfo
	err = json.Unmarshal(result, &priceInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	var skuList []*armcompute.ResourceSKU

	if strings.ToLower(productFamily) == "compute" {
		pager := priceInfoHandler.ResourceSkusClient.NewListPager(&armcompute.ResourceSKUsClientListOptions{
			// Filter: toStrPtr("location eq '" + regionName + "'"), // PriceInfo has more info than sprecific regions's ResourceSku
		})

		for pager.More() {
			page, err := pager.NextPage(priceInfoHandler.Ctx)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return "", getErr
			}

			for _, sku := range page.Value {
				skuList = append(skuList, sku)
			}
		}
	}

	organized := make(map[string][]Item)
	for _, item := range priceInfo.Items {
		organized[item.SkuID] = append(organized[item.SkuID], item)
	}

	var priceList []irs.Price
	for _, value := range organized {
		if len(value) == 0 {
			continue
		}
		// Azure Service Name: "Azure App Service", "Azure Container Apps", "Azure Kubernetes Service",
		//						"Functions", "Virtual Machines", "Cloud Services"
		if value[0].ServiceName != "Virtual Machines" {
			continue
		}

		// condition 1: check if productName contains "Windows", "Cloud Services" and "CloudServices"
		if strings.Contains(value[0].ProductName, "Windows") ||
			strings.Contains(value[0].ProductName, "Cloud Services") ||
			strings.Contains(value[0].ProductName, "CloudServices") {
			continue
		}

		// condition 2: check if skuName contains "Low Priority" or "Spot"
		if strings.Contains(value[0].SkuName, "Low Priority") || strings.Contains(value[0].SkuName, "Spot") {
			continue
		}

		productInfo := irs.ProductInfo{
			ProductId:  value[0].SkuID,
			RegionName: value[0].ArmRegionName,
			VMSpecInfo: irs.VMSpecInfo{
				Name: "NA",
				VCpu: irs.VCpuInfo{
					Count:    "-1",
					ClockGHz: "-1",
				},
				MemSizeMiB: "-1",
				DiskSizeGB: "-1",
			},
			Description:    value[0].ProductName,
			CSPProductInfo: value[0],
		}

		foundMatchingSku := false
		if strings.ToLower(productFamily) == "compute" {
			// Try to get GPU information using parseGpuInfo function
			gpuInfo := parseGpuInfo(value[0].ArmSkuName)
			if gpuInfo != nil {
				if productInfo.VMSpecInfo.Gpu == nil {
					productInfo.VMSpecInfo.Gpu = make([]irs.GpuInfo, 0)
				}
				productInfo.VMSpecInfo.Gpu = append(productInfo.VMSpecInfo.Gpu, *gpuInfo)
			}

			// Continue with the original SKU lookup logic for other VM details
			for _, sku := range skuList {
				if value[0].ArmSkuName == *sku.Name {
					foundMatchingSku = true

					for _, capability := range sku.Capabilities {
						if capability.Name == nil || capability.Value == nil {
							continue
						}

						name := *capability.Name
						value := *capability.Value

						switch name {
						case "OSVhdSizeMB":
							productInfo.VMSpecInfo.DiskSizeGB, _ = irs.ConvertMiBToGB(value)
						case "vCPUs":
							productInfo.VMSpecInfo.VCpu.Count = value
							productInfo.VMSpecInfo.VCpu.ClockGHz = "-1"
						case "MemoryGB":
							productInfo.VMSpecInfo.MemSizeMiB, _ = irs.ConvertGiBToMiB(value)
						case "GPUs":
							// Only create a default GPU entry if parseGpuInfo didn't already provide one
							if gpuInfo == nil {
								productInfo.VMSpecInfo.Gpu = []irs.GpuInfo{
									{
										Count:          value,
										MemSizeGB:      "-1",
										TotalMemSizeGB: "-1",
										Mfr:            "NA",
										Model:          "NA",
									},
								}
							}
						}
					}
					break
				}
			}

			// if no matching SKU found, skip this item
			if !foundMatchingSku {
				continue
			}

			if value[0].ArmSkuName == "" {
				productInfo.VMSpecInfo.Name = "NA"
			} else {
				productInfo.VMSpecInfo.Name = value[0].ArmSkuName
			}

			productNameToLower := strings.ToLower(value[0].ProductName)
			armSkuNameToLower := strings.ToLower(value[0].ArmSkuName)
			if strings.Contains(productNameToLower, "windows") ||
				strings.Contains(armSkuNameToLower, "windows") {
				productInfo.OSDistribution = "Windows"
			} else if strings.Contains(productNameToLower, "linux") ||
				strings.Contains(armSkuNameToLower, "linux") {
				productInfo.OSDistribution = "Linux"
			} else {
				productInfo.OSDistribution = "NA"
			}

			productInfo.PreInstalledSw = "NA"
		} else if strings.ToLower(productFamily) == "storage" {
			productInfo.VolumeType = value[0].SkuName
			productInfo.StorageMedia = "NA"
			productInfo.MaxVolumeSize = "NA"
			productInfo.MaxIOPSVolume = "NA"
			productInfo.MaxThroughputVolume = "NA"
		}

		var pricingPolicies []irs.PricingPolicies
		var isPickedByPricingPolicies bool
		isPricingPoliciesFilterExist, fields := isFieldToFilterExist(irs.PricingPolicies{}, filterList)

		for _, item := range value {
			pricingPolicy := irs.PricingPolicies{
				PricingId:     item.SkuID,
				PricingPolicy: "OnDemand",
				Unit:          strings.TrimPrefix(item.UnitOfMeasure, "1 "), // e.g. "1 Hour" -> "Hour"
				Currency:      item.CurrencyCode,
				Price:         strconv.FormatFloat(item.RetailPrice, 'f', 4, 64),
				Description:   "NA",
			}

			picked := true
			if isPricingPoliciesFilterExist {
				picked = isPicked(pricingPolicy, fields, filterList)
				if picked {
					isPickedByPricingPolicies = true
				}
			}
			if picked {
				pricingPolicies = append(pricingPolicies, pricingPolicy)
			}
		}

		// if comsuption pricing policies are not found, skip this item
		if len(pricingPolicies) == 0 {
			continue
		}

		picked := true
		isProductInfoFilterExist, fields := isFieldToFilterExist(irs.ProductInfo{}, filterList)
		if isProductInfoFilterExist {
			picked = isPicked(productInfo, fields, filterList)
		}
		if picked {
			if isPricingPoliciesFilterExist && !isPickedByPricingPolicies {
				continue
			}
			priceList = append(priceList, irs.Price{
				ProductInfo: productInfo,
				PriceInfo: irs.PriceInfo{
					PricingPolicies: pricingPolicies,
					CSPPriceInfo:    value,
				},
			})
		}
	}

	cloudPriceData := irs.CloudPriceData{
		Meta: irs.Meta{
			Version:     "v0.1",
			Description: "Multi-Cloud Price Info",
		},
		CloudPriceList: []irs.CloudPrice{
			{
				CloudName: "Azure",
				PriceList: priceList,
			},
		},
	}

	data, err := json.Marshal(cloudPriceData)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	return string(data), nil
}
