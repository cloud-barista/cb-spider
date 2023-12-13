package resources

import (
	"context"
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"io"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/url"
	"time"
)

type AzurePriceInfoHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
}

const AzurePriceApiEndpoint = "https://prices.azure.com/api/retail/prices"

type PriceInfo struct {
	BillingCurrency    string `json:"BillingCurrency"`
	CustomerEntityID   string `json:"CustomerEntityId"`
	CustomerEntityType string `json:"CustomerEntityType"`
	Items              []struct {
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
	} `json:"Items"`
	NextPageLink string `json:"NextPageLink"`
	Count        int    `json:"Count"`
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

	return string(result), nil
}
