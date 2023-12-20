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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type IbmPriceInfoHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
}

const IbmGlobalCatalogApiEndpoint = "https://globalcatalog.cloud.ibm.com/api/v1"

type Deployment struct {
	Location string `json:"location"`
}

type Pricing struct {
	URL string `json:"url"`
}

type Metadata struct {
	Deployment Deployment `json:"deployment"`
	Pricing    Pricing    `json:"pricing"`
}

type OverviewUI struct {
	EN struct {
		Description     string `json:"description"`
		DisplayName     string `json:"display_name"`
		LongDescription string `json:"long_description"`
	} `json:"en"`
}

type Resource struct {
	Id          string     `json:"id"`
	Name        string     `json:"name"`
	ChildrenURL string     `json:"children_url"`
	GeoTags     []string   `json:"geo_tags"`
	Metadata    Metadata   `json:"metadata"`
	OverviewUI  OverviewUI `json:"overview_ui"`
}

type ResourceInfo struct {
	Offset        int        `json:"offset"`
	Limit         int        `json:"limit"`
	Count         int        `json:"count"`
	ResourceCount int        `json:"resource_count"`
	First         string     `json:"first"`
	Next          string     `json:"next"`
	Resources     []Resource `json:"resources"`
}

type PriceMetric struct {
	PartRef               string `json:"part_ref"`
	MetricID              string `json:"metric_id"`
	TierModel             string `json:"tier_model"`
	ResourceDisplayName   string `json:"resource_display_name"`
	ChargeUnitDisplayName string `json:"charge_unit_display_name"`
	ChargeUnitName        string `json:"charge_unit_name"`
	ChargeUnit            string `json:"charge_unit"`
	ChargeUnitQuantity    int    `json:"charge_unit_quantity"`
	Amounts               []struct {
		Country  string `json:"country"`
		Currency string `json:"currency"`
		Prices   []struct {
			QuantityTier int     `json:"quantity_tier"`
			Price        float64 `json:"price"`
		} `json:"prices"`
	} `json:"amounts"`
	UsageCapQty    int       `json:"usage_cap_qty"`
	DisplayCap     int       `json:"display_cap"`
	EffectiveFrom  time.Time `json:"effective_from"`
	EffectiveUntil time.Time `json:"effective_until"`
}

type PriceInfo struct {
	DeploymentID       string `json:"deployment_id"`
	DeploymentLocation string `json:"deployment_location"`
	DeploymentRegion   string `json:"deployment_region"`
	Origin             string `json:"origin"`
	Type               string `json:"type"`
	I18N               struct {
	} `json:"i18n"`
	StartingPrice struct {
	} `json:"starting_price"`
	EffectiveFrom  time.Time     `json:"effective_from"`
	EffectiveUntil time.Time     `json:"effective_until"`
	Metrics        []PriceMetric `json:"metrics"`
}

func getIbmGlobalCatalog(parameters map[string]string, endpoint string) ([]byte, error) {
	ctx := context.Background()
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	q := req.URL.Query()

	for key, value := range parameters {
		q.Add(key, value)
	}

	req.URL.RawQuery = q.Encode()

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

func getIbmResourceInfo(offset int, limit int) ([]byte, error) {
	params := make(map[string]string)

	params["_offset"] = strconv.Itoa(offset)
	params["_limit"] = strconv.Itoa(limit)
	params["sort-by"] = "kind"
	params["descending"] = "true"

	return getIbmGlobalCatalog(params, IbmGlobalCatalogApiEndpoint)
}

func getIbmPlanInfo(offset int, limit int, resourceType string) ([]byte, error) {
	params := make(map[string]string)

	params["_offset"] = strconv.Itoa(offset)
	params["_limit"] = strconv.Itoa(limit)
	params["sort-by"] = "name"
	params["descending"] = "true"

	return getIbmGlobalCatalog(params, IbmGlobalCatalogApiEndpoint+"/"+resourceType+"/plan")
}

func getIbmPlanDetail(offset int, limit int, childrenURL string) ([]byte, error) {
	params := make(map[string]string)

	params["_offset"] = strconv.Itoa(offset)
	params["_limit"] = strconv.Itoa(limit)
	params["sort-by"] = "name"
	params["descending"] = "true"

	return getIbmGlobalCatalog(params, childrenURL)
}

func getIbmPriceInfo(pricingURL string) ([]byte, error) {
	params := make(map[string]string)

	return getIbmGlobalCatalog(params, pricingURL)
}

func removeDuplicateStr(array []string) []string {
	if len(array) < 1 {
		return array
	}

	sort.Strings(array)
	prev := 1
	for curr := 1; curr < len(array); curr++ {
		if array[curr-1] != array[curr] {
			array[prev] = array[curr]
			prev++
		}
	}

	return array[:prev]
}

func (priceInfoHandler *IbmPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "ListProductFamily()")
	start := call.Start()

	var resourceInfoTemp ResourceInfo

	result, err := getIbmResourceInfo(0, 1)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	err = json.Unmarshal(result, &resourceInfoTemp)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	LoggingInfo(hiscallInfo, start)

	limit := 50
	pages := resourceInfoTemp.Count / limit
	if resourceInfoTemp.Count%limit > 0 {
		pages++
	}

	var kinds []string
	var routineMax = 50
	var wait sync.WaitGroup
	var mutex = &sync.Mutex{}
	var errorOccurred bool

	for i := 0; i < pages; {
		if pages-i < routineMax {
			routineMax = pages - i
		}

		wait.Add(routineMax)

		for j := 0; j < routineMax; j++ {
			go func(wait *sync.WaitGroup, i int) {
				var rsInfoTemp ResourceInfo

				result, err = getIbmResourceInfo(limit*i, limit)
				if err != nil {
					errorOccurred = true
					wait.Done()
					return
				}

				err = json.Unmarshal(result, &rsInfoTemp)
				if err != nil {
					errorOccurred = true
					wait.Done()
					return
				}

				mutex.Lock()
				for _, resource := range rsInfoTemp.Resources {
					// Only accept name starts with 'is.'
					if strings.HasPrefix(resource.Name, "is.") {
						for _, geo := range resource.GeoTags {
							if geo == regionName {
								kinds = append(kinds, resource.Name)
							}
						}
					}
				}
				mutex.Unlock()

				wait.Done()
			}(&wait, i)

			i++
			if i == pages {
				break
			}
		}

		wait.Wait()
	}

	if errorOccurred {
		getErr := errors.New(fmt.Sprintf("Failed to List ProductFamily. err = %s",
			"Error occurred while getting ProductFamily."))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	kinds = removeDuplicateStr(kinds)

	return kinds, nil
}

func (priceInfoHandler *IbmPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "ListProductFamily()")
	start := call.Start()

	var resourceInfoTemp ResourceInfo

	result, err := getIbmPlanInfo(0, 1, productFamily)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	err = json.Unmarshal(result, &resourceInfoTemp)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	LoggingInfo(hiscallInfo, start)

	limit := 50
	pages := resourceInfoTemp.Count / limit
	if resourceInfoTemp.Count%limit > 0 {
		pages++
	}

	var resources []Resource
	var routineMax = 50
	var wait sync.WaitGroup
	var mutex = &sync.Mutex{}
	var errorOccurred bool

	for i := 0; i < pages; {
		if pages-i < routineMax {
			routineMax = pages - i
		}

		wait.Add(routineMax)

		for j := 0; j < routineMax; j++ {
			go func(wait *sync.WaitGroup, i int) {
				var rsInfoTemp ResourceInfo

				result, err = getIbmPlanInfo(limit*i, limit, productFamily)
				if err != nil {
					errorOccurred = true
					wait.Done()
					return
				}

				err = json.Unmarshal(result, &rsInfoTemp)
				if err != nil {
					errorOccurred = true
					wait.Done()
					return
				}

				mutex.Lock()
				resources = append(resources, rsInfoTemp.Resources...)
				mutex.Unlock()

				wait.Done()
			}(&wait, i)

			i++
			if i == pages {
				break
			}
		}

		wait.Wait()
	}

	if errorOccurred {
		getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s",
			"Error occurred while getting ProductFamily."))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	var priceList []irs.Price

	for _, resource := range resources {
		vCPUs := "NA"
		memoryGB := "NA"

		if strings.ToLower(productFamily) == "is.instance" {
			splitedSpec := strings.Split(resource.Name, "-")
			if len(splitedSpec) == 2 {
				splitedCPUMemory := strings.Split(splitedSpec[1], "x")
				if len(splitedCPUMemory) == 2 {
					vCPUs = splitedCPUMemory[0]
					memoryGB = splitedCPUMemory[1] + " GiB"
				}
			}
		}

		var planResourceInfoTemp ResourceInfo

		result, err := getIbmPlanDetail(0, 1, resource.ChildrenURL)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
		}

		err = json.Unmarshal(result, &planResourceInfoTemp)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
		}

		var planResources []Resource
		pages := planResourceInfoTemp.Count / limit
		if planResourceInfoTemp.Count%limit > 0 {
			pages++
		}

		for i := 0; i < pages; {
			if pages-i < routineMax {
				routineMax = pages - i
			}

			wait.Add(routineMax)

			for j := 0; j < routineMax; j++ {
				go func(wait *sync.WaitGroup, i int, routineResource Resource) {
					var planRsInfoTemp ResourceInfo

					result, err = getIbmPlanDetail(limit*i, limit, routineResource.ChildrenURL)
					if err != nil {
						errorOccurred = true
						wait.Done()
						return
					}

					err = json.Unmarshal(result, &planRsInfoTemp)
					if err != nil {
						errorOccurred = true
						wait.Done()
						return
					}

					mutex.Lock()
					planResources = append(planResources, planRsInfoTemp.Resources...)
					mutex.Unlock()

					wait.Done()
				}(&wait, i, resource)

				i++
				if i == pages {
					break
				}
			}

			wait.Wait()
		}

		var pricingURL string
		for _, planResource := range planResources {
			if planResource.Metadata.Deployment.Location == regionName {
				pricingURL = planResource.Metadata.Pricing.URL
				break
			}
		}

		if pricingURL == "" {
			continue
		}

		var priceInfo PriceInfo
		result, err = getIbmPriceInfo(pricingURL)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
		}

		err = json.Unmarshal(result, &priceInfo)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to get PriceInfo. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return "", getErr
		}

		var pricingPolicies []irs.PricingPolicies
		for _, metric := range priceInfo.Metrics {
			for _, amount := range metric.Amounts {
				if amount.Country == "USA" {
					currency := amount.Currency

					for _, price := range amount.Prices {
						pricingPolicies = append(pricingPolicies, irs.PricingPolicies{
							PricingId:         metric.MetricID,
							PricingPolicy:     "quantity_tier=" + strconv.Itoa(price.QuantityTier),
							Unit:              metric.ChargeUnit,
							Currency:          currency,
							Price:             strconv.FormatFloat(price.Price, 'f', -1, 64),
							Description:       metric.ChargeUnitDisplayName,
							PricingPolicyInfo: nil,
						})
					}
				}
			}
		}

		priceList = append(priceList, irs.Price{
			ProductInfo: irs.ProductInfo{
				ProductId:           resource.Id,
				RegionName:          regionName,
				InstanceType:        resource.Name,
				Vcpu:                vCPUs,
				Memory:              memoryGB,
				Storage:             "NA",
				Gpu:                 "NA",
				GpuMemory:           "NA",
				OperatingSystem:     "NA",
				PreInstalledSw:      "",
				VolumeType:          "NA",
				StorageMedia:        "NA",
				MaxVolumeSize:       "",
				MaxIOPSVolume:       "",
				MaxThroughputVolume: "",
				Description:         resource.OverviewUI.EN.DisplayName,
				CSPProductInfo:      resource,
			},
			PriceInfo: irs.PriceInfo{
				PricingPolicies: pricingPolicies,
				CSPPriceInfo:    priceInfo.Metrics,
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
				CloudName: "IBM",
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
