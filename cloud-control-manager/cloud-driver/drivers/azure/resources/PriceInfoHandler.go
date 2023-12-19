package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type AzurePriceInfoHandler struct {
	CredentialInfo     idrv.CredentialInfo
	Region             idrv.RegionInfo
	Ctx                context.Context
	ResourceSkusClient *compute.ResourceSkusClient
}

const AzurePriceApiEndpoint = "https://prices.azure.com/api/retail/prices"

type Item struct {
	CurrencyCode         string    `json:"currencyCode"`
	TierMinimumUnits     float64   `json:"tierMinimumUnits"`
	RetailPrice          float64   `json:"retailPrice"`
	UnitPrice            float64   `json:"unitPrice"`
	ArmRegionName        string    `json:"armRegionName"`
	Location             string    `json:"location"`
	EffectiveStartDate   time.Time `json:"effectiveStartDate"`
	MeterID              string    `json:"meterId"`
	MeterName            string    `json:"meterName"`
	ProductID            string    `json:"productId"`
	SkuID                string    `json:"skuId"`
	ProductName          string    `json:"productName"`
	SkuName              string    `json:"skuName"`
	ServiceName          string    `json:"serviceName"`
	ServiceID            string    `json:"serviceId"`
	ServiceFamily        string    `json:"serviceFamily"`
	UnitOfMeasure        string    `json:"unitOfMeasure"`
	Type                 string    `json:"type"`
	IsPrimaryMeterRegion bool      `json:"isPrimaryMeterRegion"`
	ArmSkuName           string    `json:"armSkuName"`
	ReservationTerm      string    `json:"reservationTerm,omitempty"`
	EffectiveEndDate     time.Time `json:"effectiveEndDate,omitempty"`
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

	fmt.Println(URL)

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
	for _, filter := range filterList {
		filterOption += " and " + filter.Key + " eq '" + filter.Value + "'"
	}

	result, err := getAzurePriceInfo(filterOption)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	LoggingInfo(hiscallInfo, start)

	var priceInfo PriceInfo
	err = json.Unmarshal(result, &priceInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	var resultResourceSkusClient compute.ResourceSkusResultPage

	if strings.ToLower(productFamily) == "compute" {
		resultResourceSkusClient, err = priceInfoHandler.ResourceSkusClient.List(priceInfoHandler.Ctx, "location eq '"+regionName+"'")
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
		}
	}

	organized := make(map[string][]Item)
	for _, item := range priceInfo.Items {
		organized[item.ProductID] = append(organized[item.ProductID], item)
	}

	var priceList []irs.PriceList
	for _, value := range organized {
		if len(value) == 0 {
			continue
		}

		vCPUs := "NA"
		memoryGB := "NA"
		storageGB := "NA"
		operatingSystem := "NA"

		if strings.ToLower(productFamily) == "compute" {
			for _, val := range resultResourceSkusClient.Values() {
				if value[0].SkuName == *val.Name {
					for _, capability := range *val.Capabilities {
						if *capability.Name == "OSVhdSizeMB" {
							sizeMB, _ := strconv.Atoi(*capability.Value)
							sizeGB := float64(sizeMB) / 1024
							storageGB = strconv.FormatFloat(sizeGB, 'f', -1, 64) + " GiB"
						} else if *capability.Name == "vCPUs" {
							vCPUs = *capability.Value
						} else if *capability.Name == "MemoryGB" {
							memoryGB = *capability.Value + " GiB"
						}
					}
				}
			}
		}

		productNameToLower := strings.ToLower(value[0].ProductName)
		if strings.Contains(productNameToLower, "windows") {
			operatingSystem = "Windows"
		} else if strings.Contains(productNameToLower, "linux") {
			operatingSystem = "Linux"
		}

		var pricingPolicies []irs.PricingPolicies
		for _, item := range value {
			pricingPolicies = append(pricingPolicies, irs.PricingPolicies{
				PricingId:         item.SkuID,
				PricingPolicy:     item.Type,
				Unit:              item.UnitOfMeasure,
				Currency:          item.CurrencyCode,
				Price:             strconv.FormatFloat(item.RetailPrice, 'f', -1, 64),
				Description:       "NA",
				PricingPolicyInfo: nil,
			})
		}

		priceList = append(priceList, irs.PriceList{
			ProductInfo: irs.ProductInfo{
				ProductId:           value[0].ProductID,
				RegionName:          value[0].ArmRegionName,
				InstanceType:        value[0].ArmSkuName,
				Vcpu:                vCPUs,
				Memory:              memoryGB,
				Storage:             storageGB,
				Gpu:                 "NA",
				GpuMemory:           "NA",
				OperatingSystem:     operatingSystem,
				PreInstalledSw:      "",
				VolumeType:          "NA",
				StorageMedia:        "NA",
				MaxVolumeSize:       "",
				MaxIOPSVolume:       "",
				MaxThroughputVolume: "",
				Description:         value[0].ProductName,
				CSPProductInfo:      value[0],
			},
			PriceInfo: irs.PriceInfo{
				PricingPolicies: pricingPolicies,
				CSPPriceInfo:    value,
			},
		})
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
		getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	file, err := os.OpenFile("1", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = file.WriteString(string(data))
	if err != nil {
		return "", err
	}

	return string(data), nil
}
