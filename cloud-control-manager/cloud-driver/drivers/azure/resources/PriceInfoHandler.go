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

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzurePriceInfoHandler struct {
	CredentialInfo     idrv.CredentialInfo
	Region             idrv.RegionInfo
	Ctx                context.Context
	ResourceSkusClient *compute.ResourceSkusClient
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

	fmt.Println(url.Parse(URL))

	ctx := context.Background()
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return responseBody, nil
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

	filterOption := "serviceFamily eq '" + productFamily + "' and armRegionName eq '" + regionName + "'"

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

	var resultResourceSkusClient compute.ResourceSkusResultPage

	if strings.ToLower(productFamily) == "compute" {
		resultResourceSkusClient, err = priceInfoHandler.ResourceSkusClient.List(priceInfoHandler.Ctx, "location eq '"+regionName+"'")
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
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

		productInfo := irs.ProductInfo{
			ProductId:      value[0].ProductID,
			RegionName:     value[0].ArmRegionName,
			Description:    value[0].ProductName,
			CSPProductInfo: value[0],
		}

		if strings.ToLower(productFamily) == "compute" {
			for _, val := range resultResourceSkusClient.Values() {
				if value[0].SkuName == *val.Name {
					for _, capability := range *val.Capabilities {
						if *capability.Name == "OSVhdSizeMB" {
							sizeMB, _ := strconv.Atoi(*capability.Value)
							sizeGB := float64(sizeMB) / 1024
							productInfo.Storage = strconv.FormatFloat(sizeGB, 'f', -1, 64) + " GiB"
						} else if *capability.Name == "vCPUs" {
							productInfo.Vcpu = *capability.Value
						} else if *capability.Name == "MemoryGB" {
							productInfo.Memory = *capability.Value + " GiB"
						}
					}
				}
			}

			productInfo.InstanceType = value[0].ArmSkuName

			productNameToLower := strings.ToLower(value[0].ProductName)
			armSkuNameToLower := strings.ToLower(value[0].ArmSkuName)
			if strings.Contains(productNameToLower, "windows") ||
				strings.Contains(armSkuNameToLower, "windows") {
				productInfo.OperatingSystem = "Windows"
			} else if strings.Contains(productNameToLower, "linux") ||
				strings.Contains(armSkuNameToLower, "linux") {
				productInfo.OperatingSystem = "Linux"
			} else {
				productInfo.OperatingSystem = "NA"
			}

			productInfo.Gpu = "NA"
			productInfo.GpuMemory = "NA"
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
				PricingPolicy: item.Type,
				Unit:          item.UnitOfMeasure,
				Currency:      item.CurrencyCode,
				Price:         strconv.FormatFloat(item.RetailPrice, 'f', -1, 64),
				Description:   "NA",
				PricingPolicyInfo: &irs.PricingPolicyInfo{
					LeaseContractLength: "NA",
					OfferingClass:       "NA",
					PurchaseOption:      "NA",
				},
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
