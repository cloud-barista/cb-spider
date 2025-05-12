package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"k8s.io/apimachinery/pkg/util/json"
)

type IbmPriceInfoHandler struct {
	Region         idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
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
	timeout := 300 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

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

func isFieldToFilterExist(structVal any, filterList []irs.KeyValue) (exist bool, fields []string) {
	var val reflect.Value

	if len(filterList) == 0 {
		return false, fields
	}

	if _, ok := structVal.(irs.ProductInfo); ok {
		data := structVal.(irs.ProductInfo)
		val = reflect.ValueOf(&data).Elem()
	} else if _, ok := structVal.(irs.OnDemand); ok {
		data := structVal.(irs.OnDemand)
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
	} else if _, ok := structVal.(irs.OnDemand); ok {
		data := structVal.(irs.OnDemand)
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
					if strings.HasPrefix(resource.Name, "is.") || resource.Name == "containers-kubernetes" {
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

type VPCProfile struct {
	Name         string `json:"name"`
	Family       string `json:"family"`
	VCPU         int    `json:"vcpu"`
	Memory       int    `json:"memory"`
	GPUModel     string `json:"gpu_model,omitempty"`
	GPUCount     int    `json:"gpu_count,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Generation   string `json:"generation"`
}

type ServicePlan struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ID          string `json:"id"`
	ChildrenURL string `json:"children_url"`
}

type ProfilePricing struct {
	ProfileName  string
	RegionName   string
	MetricPrices map[string]struct {
		Price    float64
		Currency string
		Unit     string
	}
}

type ExtendedPricingInfo struct {
	Metrics []PriceMetric `json:"metrics"`
	Origin  string        `json:"origin"`
	Type    string        `json:"type"`
}

type ExtendedMetadata struct {
	Deployment struct {
		Location string `json:"location"`
	} `json:"deployment"`
	Pricing      ExtendedPricingInfo `json:"pricing"`
	RcCompatible bool                `json:"rc_compatible"`
}

type ExtendedResource struct {
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	ChildrenURL string           `json:"children_url"`
	GeoTags     []string         `json:"geo_tags"`
	Metadata    ExtendedMetadata `json:"metadata"`
	OverviewUI  OverviewUI       `json:"overview_ui"`
}

type ExtendedResourceInfo struct {
	Offset        int                `json:"offset"`
	Limit         int                `json:"limit"`
	Count         int                `json:"count"`
	ResourceCount int                `json:"resource_count"`
	First         string             `json:"first"`
	Next          string             `json:"next"`
	Resources     []ExtendedResource `json:"resources"`
}

func (priceInfoHandler *IbmPriceInfoHandler) GetProfilesForRegion(regionName string) ([]VPCProfile, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "GetProfilesForRegion()")
	start := call.Start()

	vmSpecHandler := &IbmVmSpecHandler{
		CredentialInfo: priceInfoHandler.CredentialInfo,
		Region:         priceInfoHandler.Region,
		VpcService:     priceInfoHandler.VpcService,
		Ctx:            priceInfoHandler.Ctx,
	}

	specList, err := vmSpecHandler.ListVMSpec()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to list instance profiles. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	var profiles []VPCProfile
	for _, spec := range specList {
		vcpu, err := strconv.Atoi(spec.VCpu.Count)
		if err != nil {
			cblogger.Error(fmt.Sprintf("Failed to parse vCPU count: %s", err))
			vcpu = 1
		}

		memory := 1
		if spec.MemSizeMiB != "" && spec.MemSizeMiB != "-1" {
			memMiB, err := strconv.Atoi(spec.MemSizeMiB)
			if err == nil {
				memory = memMiB / 1024
				if memory < 1 {
					memory = 1
				}
			}
		}

		gpuModel := ""
		gpuCount := 0
		if len(spec.Gpu) > 0 {
			for _, gpu := range spec.Gpu {
				if gpu.Model != "NA" {
					gpuModel = gpu.Model
					if gpu.Count != "-1" {
						count, err := strconv.Atoi(gpu.Count)
						if err == nil {
							gpuCount = count
						}
					}
					break
				}
			}
		}

		generation := "unknown"
		if isGen2Profile(spec.Name) {
			generation = "gen2"
		} else if strings.Contains(spec.Name, "x3") || strings.Contains(spec.Name, "-gen3") {
			generation = "gen3"
		}

		architecture := "amd64"
		family := ""

		for _, kv := range spec.KeyValueList {
			if kv.Key == "Architecture" {
				architecture = kv.Value
			} else if kv.Key == "Family" {
				family = kv.Value
			}
		}

		if strings.Contains(spec.Name, "z2") && architecture == "amd64" {
			architecture = "s390x"
		}

		if family == "" {
			if parts := strings.Split(spec.Name, "-"); len(parts) > 0 {
				firstPart := parts[0]
				if len(firstPart) > 0 {
					switch firstPart[0] {
					case 'b':
						family = "balanced"
					case 'c':
						family = "compute"
					case 'm':
						family = "memory"
					case 'g':
						family = "gpu"
					case 'o':
						family = "storage_optimized"
					default:
						family = "unknown"
					}

					if strings.Contains(firstPart, "z2") {
						family = family + "_z"
					}
				}
			}
		}

		profiles = append(profiles, VPCProfile{
			Name:         spec.Name,
			Family:       family,
			VCPU:         vcpu,
			Memory:       memory,
			GPUModel:     gpuModel,
			GPUCount:     gpuCount,
			Architecture: architecture,
			Generation:   generation,
		})
	}

	LoggingInfo(hiscallInfo, start)
	return profiles, nil
}

func (priceInfoHandler *IbmPriceInfoHandler) GetServicePlans(productFamily string) ([]ServicePlan, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "GetServicePlans()")
	start := call.Start()

	result, err := getIbmPlanInfo(0, 1, productFamily)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get Service Plans. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	var resourceInfoTemp ResourceInfo
	err = json.Unmarshal(result, &resourceInfoTemp)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to unmarshal Service Plans. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	totalCount := resourceInfoTemp.Count
	limit := 50
	pages := totalCount / limit
	if totalCount%limit > 0 {
		pages++
	}

	cblogger.Info(fmt.Sprintf("Found %d total service plans for %s", totalCount, productFamily))

	var servicePlans []ServicePlan
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

				pageResult, err := getIbmPlanInfo(limit*i, limit, productFamily)
				if err != nil {
					errorOccurred = true
					wait.Done()
					return
				}

				err = json.Unmarshal(pageResult, &rsInfoTemp)
				if err != nil {
					errorOccurred = true
					wait.Done()
					return
				}

				var plans []ServicePlan
				for _, resource := range rsInfoTemp.Resources {
					plans = append(plans, ServicePlan{
						Name:        resource.Name,
						Description: resource.OverviewUI.EN.Description,
						ID:          resource.Id,
						ChildrenURL: resource.ChildrenURL,
					})
				}

				mutex.Lock()
				servicePlans = append(servicePlans, plans...)
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
		getErr := errors.New(fmt.Sprintf("Error occurred while retrieving service plans"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	LoggingInfo(hiscallInfo, start)
	return servicePlans, nil
}

func (priceInfoHandler *IbmPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "GetPriceInfo()")
	start := call.Start()

	profiles, err := priceInfoHandler.GetProfilesForRegion(regionName)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get region profiles. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	cblogger.Info(fmt.Sprintf("Retrieved %d profiles for region %s", len(profiles), regionName))

	servicePlans, err := priceInfoHandler.GetServicePlans(productFamily)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get service plans. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	cblogger.Info(fmt.Sprintf("Retrieved %d service plans for product family %s", len(servicePlans), productFamily))

	var priceList []irs.Price

	if strings.ToLower(productFamily) == "is.instance" {
		var gen2Plan *ServicePlan
		for i, plan := range servicePlans {
			if plan.Name == "gen2-instance" {
				gen2Plan = &servicePlans[i]
				break
			}
		}

		if gen2Plan != nil {
			cblogger.Info(fmt.Sprintf("Found Gen2 service plan: %s", gen2Plan.Name))

			var gen2Profiles []VPCProfile
			for _, profile := range profiles {
				if profile.Generation == "gen2" {
					gen2Profiles = append(gen2Profiles, profile)
				}
			}
			cblogger.Info(fmt.Sprintf("Found %d Gen2 profiles to get pricing for", len(gen2Profiles)))

			gen2ProfilePrices, err := priceInfoHandler.GetGen2ProfilePrices(gen2Plan.ChildrenURL, regionName, gen2Profiles)
			if err != nil {
				cblogger.Error(fmt.Sprintf("Error getting Gen2 profile prices: %s", err.Error()))
			} else {
				for _, profilePricing := range gen2ProfilePrices {
					var profile *VPCProfile
					for i, p := range gen2Profiles {
						if p.Name == profilePricing.ProfileName {
							profile = &gen2Profiles[i]
							break
						}
					}

					if profile != nil {
						price, err := priceInfoHandler.CreateGen2ProfilePrice(*profile, profilePricing, regionName, filterList)
						if err != nil {
							cblogger.Warning(fmt.Sprintf("Error creating price for Gen2 profile %s: %s", profile.Name, err.Error()))
							continue
						}
						priceList = append(priceList, price)
						cblogger.Info(fmt.Sprintf("Added Gen2 profile price for %s", profile.Name))
					}
				}
				cblogger.Info(fmt.Sprintf("Added %d Gen2 profile prices to the list", len(gen2ProfilePrices)))
			}
		} else {
			cblogger.Warning("Gen2 service plan not found")
		}

		var gen3Profiles []VPCProfile
		for _, profile := range profiles {
			if profile.Generation != "gen2" {
				gen3Profiles = append(gen3Profiles, profile)
			}
		}
		cblogger.Info(fmt.Sprintf("Found %d Gen3/other profiles to process", len(gen3Profiles)))

		for _, profile := range gen3Profiles {
			var matchingPlan *ServicePlan
			for i, plan := range servicePlans {
				if plan.Name == profile.Name {
					matchingPlan = &servicePlans[i]
					break
				}
			}

			if matchingPlan != nil {
				cblogger.Info(fmt.Sprintf("Found matching service plan for profile %s", profile.Name))

				price, err := priceInfoHandler.GetGen3ProfilePrice(matchingPlan.ChildrenURL, profile, regionName, filterList)
				if err != nil {
					cblogger.Warning(fmt.Sprintf("Could not get price for profile %s: %s", profile.Name, err.Error()))
					continue
				}

				priceList = append(priceList, price)
				cblogger.Info(fmt.Sprintf("Added price for profile %s to price list", profile.Name))
			} else {
				cblogger.Warning(fmt.Sprintf("No matching service plan found for profile %s", profile.Name))
			}
		}
	}

	cloudPrice := irs.CloudPrice{
		Meta:       irs.Meta{Version: "0.5", Description: "IBM VPC Virtual Machines Price Info"},
		CloudName:  "IBM",
		RegionName: regionName,
		ZoneName:   "NA",
		PriceList:  priceList,
	}

	data, err := json.Marshal(cloudPrice)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to marshal price data. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	LoggingInfo(hiscallInfo, start)
	return string(data), nil
}

func (priceInfoHandler *IbmPriceInfoHandler) GetGen2ProfilePrices(childrenURL string, regionName string, gen2Profiles []VPCProfile) ([]ProfilePricing, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "GetGen2ProfilePrices()")
	start := call.Start()

	initialResponse, err := getIbmPlanDetail(0, 1, childrenURL)
	if err != nil {
		return nil, err
	}

	var initialDeployments ResourceInfo
	err = json.Unmarshal(initialResponse, &initialDeployments)
	if err != nil {
		return nil, err
	}

	totalResources := initialDeployments.Count
	cblogger.Info(fmt.Sprintf("Found %d total deployment resources for Gen2 profiles", totalResources))

	limit := 100
	pages := totalResources / limit
	if totalResources%limit > 0 {
		pages++
	}

	var allResources []ExtendedResource
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
				offset := i * limit
				response, err := getIbmPlanDetail(offset, limit, childrenURL)
				if err != nil {
					errorOccurred = true
					wait.Done()
					return
				}

				var pageDeployments ExtendedResourceInfo
				err = json.Unmarshal(response, &pageDeployments)
				if err != nil {
					errorOccurred = true
					wait.Done()
					return
				}

				mutex.Lock()
				allResources = append(allResources, pageDeployments.Resources...)
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
		return nil, errors.New("Error retrieving Gen2 deployment resources")
	}

	cblogger.Info(fmt.Sprintf("Retrieved %d total deployment resources", len(allResources)))

	var regionResources []ExtendedResource

	profileNameRegex := regexp.MustCompile(`^([a-z0-9-]+)-` + regionName + "$")

	for _, res := range allResources {
		matched := false

		matches := profileNameRegex.FindStringSubmatch(res.Name)
		if len(matches) > 1 {
			profileName := matches[1]

			for _, profile := range gen2Profiles {
				if profile.Name == profileName {
					cblogger.Info(fmt.Sprintf("Found matching resource for profile: %s", profile.Name))
					regionResources = append(regionResources, res)
					matched = true
					break
				}
			}

			if matched {
				continue
			}
		}

		for _, profile := range gen2Profiles {
			if (strings.Contains(res.Name, "-"+profile.Name+"-") ||
				strings.HasPrefix(res.Name, profile.Name+"-") ||
				strings.Contains(res.Name, "profile-"+profile.Name)) &&
				(res.Metadata.Deployment.Location == regionName ||
					containsRegionInGeoTags(res.GeoTags, regionName)) {

				cblogger.Info(fmt.Sprintf("Found resource with profile name in it: %s for profile %s",
					res.Name, profile.Name))
				regionResources = append(regionResources, res)
				matched = true
				break
			}
		}

		if matched {
			continue
		}

		for _, profile := range gen2Profiles {
			if strings.Contains(profile.Name, "z2") {
				if (strings.Contains(res.Name, "ibmz") ||
					strings.Contains(res.Name, "s390x") ||
					strings.Contains(res.Name, "z-arch")) &&
					(res.Metadata.Deployment.Location == regionName ||
						containsRegionInGeoTags(res.GeoTags, regionName)) {

					cblogger.Info(fmt.Sprintf("Found Z architecture resource: %s for profile %s",
						res.Name, profile.Name))
					regionResources = append(regionResources, res)
					matched = true
					break
				}
			}
		}

		if matched {
			continue
		}

		if res.Metadata.Deployment.Location == regionName ||
			containsRegionInGeoTags(res.GeoTags, regionName) {
			cblogger.Info(fmt.Sprintf("Found resource for region by deployment location/geo_tags: %s", res.Name))
			regionResources = append(regionResources, res)
		}
	}

	cblogger.Info(fmt.Sprintf("Found %d resources matching region %s", len(regionResources), regionName))

	var profilePricings []ProfilePricing
	targetChargeUnits := []string{
		"MEMORY_HOURS",
		"VCPU_HOURS",
		"IS_STORAGE_GIGABYTE_HOURS",
		"IS_LARGE_STORAGE_GIGABYTE_HOURS",
		"V100_HOURS",
		"SECUREEXECUTION_VCPU_HOURS",
		"SECUREEXECUTION_MEMORY_HOURS",
		"Z_VCPU_HOURS",
		"Z_MEMORY_HOURS",
		"Z_STORAGE_GIGABYTE_HOURS",
	}

	profilePatterns := make(map[string][]string)

	for _, profile := range gen2Profiles {
		patterns := []string{
			profile.Name,
			profile.Name + "-" + regionName,
		}

		if strings.Contains(profile.Name, "x2") {
			baseName := strings.Replace(profile.Name, "d", "", -1)
			if baseName != profile.Name {
				patterns = append(patterns, baseName)
			}
		}

		if strings.Contains(profile.Name, "z2") {
			baseName := strings.Replace(profile.Name, "e", "", -1)
			if baseName != profile.Name {
				patterns = append(patterns, baseName)
			}

			patterns = append(patterns, "ibmz", "s390x", "z-arch")
		}

		profilePatterns[profile.Name] = patterns
	}

	for _, res := range regionResources {
		matchedProfileName := ""
		highestMatchScore := 0

		for profileName, patterns := range profilePatterns {
			for _, pattern := range patterns {
				if strings.Contains(res.Name, pattern) {
					matchScore := len(pattern)
					if matchScore > highestMatchScore {
						highestMatchScore = matchScore
						matchedProfileName = profileName
					}
				}
			}
		}

		if matchedProfileName == "" {
			matches := profileNameRegex.FindStringSubmatch(res.Name)
			if len(matches) > 1 {
				baseProfileName := matches[1]

				for _, profile := range gen2Profiles {
					if strings.HasPrefix(profile.Name, baseProfileName) ||
						strings.HasPrefix(baseProfileName, profile.Name) {
						matchedProfileName = profile.Name
						break
					}
				}
			}

			if matchedProfileName == "" {
				cblogger.Warning(fmt.Sprintf("Could not determine profile for resource: %s", res.Name))
				continue
			}
		}

		cblogger.Info(fmt.Sprintf("Processing pricing for profile: %s from resource: %s",
			matchedProfileName, res.Name))

		if len(res.Metadata.Pricing.Metrics) == 0 {
			cblogger.Warning(fmt.Sprintf("Resource %s has no pricing metrics", res.Name))
			continue
		}

		metricPrices := make(map[string]struct {
			Price    float64
			Currency string
			Unit     string
		})

		for _, metric := range res.Metadata.Pricing.Metrics {
			chargeUnitName := metric.ChargeUnitName

			isTargetUnit := false
			for _, target := range targetChargeUnits {
				if chargeUnitName == target {
					isTargetUnit = true
					break
				}
			}

			if !isTargetUnit {
				continue
			}

			var bestPrice struct {
				Price    float64
				Currency string
			}

			hasUSAPrice := false
			for _, amount := range metric.Amounts {
				if amount.Country == "USA" && amount.Currency == "USD" && len(amount.Prices) > 0 {
					bestPrice.Price = amount.Prices[0].Price
					bestPrice.Currency = amount.Currency
					hasUSAPrice = true
					break
				}
			}

			if !hasUSAPrice && len(metric.Amounts) > 0 && len(metric.Amounts[0].Prices) > 0 {
				bestPrice.Price = metric.Amounts[0].Prices[0].Price
				bestPrice.Currency = metric.Amounts[0].Currency
			}

			if bestPrice.Currency != "" {
				metricPrices[chargeUnitName] = struct {
					Price    float64
					Currency string
					Unit     string
				}{
					Price:    bestPrice.Price,
					Currency: bestPrice.Currency,
					Unit:     metric.ChargeUnit,
				}

				cblogger.Info(fmt.Sprintf("Found price for %s: %f %s per %s",
					chargeUnitName, bestPrice.Price, bestPrice.Currency, metric.ChargeUnit))
			}
		}

		if len(metricPrices) > 0 {
			existingIndex := -1
			for i, pp := range profilePricings {
				if pp.ProfileName == matchedProfileName {
					existingIndex = i
					break
				}
			}

			if existingIndex >= 0 {
				for k, v := range metricPrices {
					profilePricings[existingIndex].MetricPrices[k] = v
				}
				cblogger.Info(fmt.Sprintf("Updated pricing for profile %s with %d additional metrics",
					matchedProfileName, len(metricPrices)))
			} else {
				profilePricings = append(profilePricings, ProfilePricing{
					ProfileName:  matchedProfileName,
					RegionName:   regionName,
					MetricPrices: metricPrices,
				})
				cblogger.Info(fmt.Sprintf("Extracted pricing for profile %s in region %s with %d metrics",
					matchedProfileName, regionName, len(metricPrices)))
			}
		} else {
			cblogger.Warning(fmt.Sprintf("No pricing metrics found for profile %s", matchedProfileName))
		}
	}

	missingZProfiles := make(map[string]string)

	for _, profile := range gen2Profiles {
		if strings.Contains(profile.Name, "z2e") {
			found := false
			for _, pp := range profilePricings {
				if pp.ProfileName == profile.Name {
					found = true
					break
				}
			}

			if !found {
				baseProfileName := strings.Replace(profile.Name, "e", "", -1)
				missingZProfiles[profile.Name] = baseProfileName
			}
		}
	}

	for missingProfile, baseProfile := range missingZProfiles {
		for _, pp := range profilePricings {
			if pp.ProfileName == baseProfile {
				newPricing := ProfilePricing{
					ProfileName: missingProfile,
					RegionName:  regionName,
					MetricPrices: make(map[string]struct {
						Price    float64
						Currency string
						Unit     string
					}),
				}

				for k, v := range pp.MetricPrices {
					newPricing.MetricPrices[k] = v
				}

				profilePricings = append(profilePricings, newPricing)
				cblogger.Info(fmt.Sprintf("Created pricing for missing Z profile %s based on %s with %d metrics",
					missingProfile, baseProfile, len(newPricing.MetricPrices)))
				break
			}
		}
	}

	LoggingInfo(hiscallInfo, start)
	return profilePricings, nil
}

func (priceInfoHandler *IbmPriceInfoHandler) CreateGen2ProfilePrice(profile VPCProfile, profilePricing ProfilePricing, regionName string, filterList []irs.KeyValue) (irs.Price, error) {
	isIBMZ := profile.Architecture == "s390x" || strings.Contains(profile.Name, "z2")

	var description string
	if isIBMZ {
		description = fmt.Sprintf("IBM VPC Gen2 IBM Z instance profile %s with %d vCPU and %d GB RAM",
			profile.Name, profile.VCPU, profile.Memory)
	} else {
		description = fmt.Sprintf("IBM VPC Gen2 instance profile %s with %d vCPU and %d GB RAM",
			profile.Name, profile.VCPU, profile.Memory)
	}

	productInfo := irs.ProductInfo{
		ProductId:      fmt.Sprintf("%s-%s", profile.Name, regionName),
		Description:    description,
		CSPProductInfo: profile,
	}

	vmSpecInfo := irs.VMSpecInfo{
		Region:     regionName,
		Name:       profile.Name,
		VCpu:       irs.VCpuInfo{Count: strconv.Itoa(profile.VCPU), ClockGHz: "-1"},
		MemSizeMiB: strconv.Itoa(profile.Memory * 1024),
		DiskSizeGB: "-1",
	}

	if profile.GPUCount > 0 && profile.GPUModel != "" {
		gpuMfr := "NVIDIA"
		vmSpecInfo.Gpu = []irs.GpuInfo{
			{
				Count:          strconv.Itoa(profile.GPUCount),
				Mfr:            gpuMfr,
				Model:          profile.GPUModel,
				MemSizeGB:      "-1",
				TotalMemSizeGB: "-1",
			},
		}
	} else {
		vmSpecInfo.Gpu = []irs.GpuInfo{}
	}

	vmSpecInfo.KeyValueList = []irs.KeyValue{
		{Key: "Family", Value: profile.Family},
		{Key: "Architecture", Value: profile.Architecture},
		{Key: "Generation", Value: profile.Generation},
	}

	if isIBMZ {
		vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList,
			irs.KeyValue{Key: "Platform", Value: "IBM Z"})

		if strings.HasSuffix(profile.Name, "e") {
			vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList,
				irs.KeyValue{Key: "Secure", Value: "Enhanced"})
		}
	}

	productInfo.VMSpecInfo = vmSpecInfo

	var totalPrice float64

	var allPriceDetails []map[string]interface{}
	for chargeUnitName, priceData := range profilePricing.MetricPrices {
		var price float64
		var description string

		switch chargeUnitName {
		case "MEMORY_HOURS":
			price = priceData.Price
			description = fmt.Sprintf("Memory-Hours (%d GB)", profile.Memory)
		case "VCPU_HOURS":
			price = priceData.Price
			description = fmt.Sprintf("vCPU-Hours (%d vCPU)", profile.VCPU)
		case "IS_STORAGE_GIGABYTE_HOURS":
			price = priceData.Price
			description = "Storage (Standard) Gigabyte-Hours"
		case "IS_LARGE_STORAGE_GIGABYTE_HOURS":
			price = priceData.Price
			description = "Storage (Large) Gigabyte-Hours"
		case "V100_HOURS":
			price = priceData.Price
			description = fmt.Sprintf("GPU-Hours (%d V100)", profile.GPUCount)
		case "SECUREEXECUTION_VCPU_HOURS":
			price = priceData.Price
			description = fmt.Sprintf("Secure Execution vCPU-Hours (%d vCPU)", profile.VCPU)
		case "SECUREEXECUTION_MEMORY_HOURS":
			price = priceData.Price
			description = fmt.Sprintf("Secure Execution Memory-Hours (%d GB)", profile.Memory)
		case "Z_VCPU_HOURS":
			price = priceData.Price
			description = fmt.Sprintf("IBM Z vCPU-Hours (%d vCPU)", profile.VCPU)
		case "Z_MEMORY_HOURS":
			price = priceData.Price
			description = fmt.Sprintf("IBM Z Memory-Hours (%d GB)", profile.Memory)
		case "Z_STORAGE_GIGABYTE_HOURS":
			price = priceData.Price
			description = "IBM Z Storage Gigabyte-Hours"
		default:
			price = priceData.Price
			description = chargeUnitName
		}

		totalPrice += price

		priceDetail := map[string]interface{}{
			"chargeUnitName": chargeUnitName,
			"price":          price,
			"currency":       priceData.Currency,
			"unit":           priceData.Unit,
			"description":    description,
		}
		allPriceDetails = append(allPriceDetails, priceDetail)
	}

	var onDemand irs.OnDemand

	if len(allPriceDetails) > 0 {
		var totalDescription string
		if isIBMZ {
			totalDescription = fmt.Sprintf("Total hourly cost for IBM Z profile %s (%d vCPU, %d GB)",
				profile.Name, profile.VCPU, profile.Memory)
		} else {
			totalDescription = fmt.Sprintf("Total hourly cost for %s (%d vCPU, %d GB)",
				profile.Name, profile.VCPU, profile.Memory)
		}

		onDemand = irs.OnDemand{
			PricingId:   "TOTAL",
			Unit:        "Hour",
			Currency:    "USD",
			Price:       strconv.FormatFloat(math.Round(totalPrice*1000)/1000, 'f', 3, 64),
			Description: totalDescription,
		}
	}

	matchesFilter := true
	isOnDemandFilterExist, fields := isFieldToFilterExist(irs.OnDemand{}, filterList)

	if isOnDemandFilterExist && onDemand.Price != "" {
		if !isPicked(onDemand, fields, filterList) {
			matchesFilter = false
		}
	}

	isProductInfoFilterExist, fields := isFieldToFilterExist(irs.ProductInfo{}, filterList)
	if isProductInfoFilterExist && !isPicked(productInfo, fields, filterList) {
		matchesFilter = false
	}

	if !matchesFilter || onDemand.Price == "" {
		return irs.Price{}, fmt.Errorf("Profile %s doesn't match filters or has no pricing information", profile.Name)
	}

	priceInfo := irs.PriceInfo{
		OnDemand: onDemand,
		CSPPriceInfo: map[string]interface{}{
			"profilePricing": profilePricing,
			"priceDetails":   allPriceDetails,
		},
	}

	return irs.Price{
		ProductInfo: productInfo,
		PriceInfo:   priceInfo,
	}, nil
}

func (priceInfoHandler *IbmPriceInfoHandler) GetGen3ProfilePrice(childrenURL string, profile VPCProfile, regionName string, filterList []irs.KeyValue) (irs.Price, error) {
	var price irs.Price

	initialResponse, err := getIbmPlanDetail(0, 1, childrenURL)
	if err != nil {
		return price, err
	}

	var initialDeployments ResourceInfo
	err = json.Unmarshal(initialResponse, &initialDeployments)
	if err != nil {
		return price, err
	}

	totalResources := initialDeployments.Count
	cblogger.Info(fmt.Sprintf("Found %d deployment resources for profile %s", totalResources, profile.Name))

	limit := 100
	pages := totalResources / limit
	if totalResources%limit > 0 {
		pages++
	}

	var allResources []Resource
	for i := 0; i < pages; i++ {
		offset := i * limit
		response, err := getIbmPlanDetail(offset, limit, childrenURL)
		if err != nil {
			return price, err
		}

		var pageDeployments ResourceInfo
		err = json.Unmarshal(response, &pageDeployments)
		if err != nil {
			return price, err
		}

		allResources = append(allResources, pageDeployments.Resources...)
	}

	var pricingURL string
	for _, resource := range allResources {
		if resource.Metadata.Deployment.Location == regionName ||
			containsRegionInGeoTags(resource.GeoTags, regionName) {
			if resource.Metadata.Pricing.URL != "" {
				pricingURL = resource.Metadata.Pricing.URL
				cblogger.Info(fmt.Sprintf("Found pricing URL for profile %s in region %s", profile.Name, regionName))
				break
			}
		}
	}

	if pricingURL == "" {
		return price, fmt.Errorf("No pricing URL found for profile %s in region %s", profile.Name, regionName)
	}

	response, err := getIbmPriceInfo(pricingURL)
	if err != nil {
		return price, err
	}

	var priceInfo PriceInfo
	err = json.Unmarshal(response, &priceInfo)
	if err != nil {
		return price, err
	}

	productInfo := irs.ProductInfo{
		ProductId:      fmt.Sprintf("%s-%s", profile.Name, regionName),
		Description:    fmt.Sprintf("IBM VPC %s instance profile %s with %d vCPU and %d GB RAM", profile.Generation, profile.Name, profile.VCPU, profile.Memory),
		CSPProductInfo: profile,
	}

	vmSpecInfo := irs.VMSpecInfo{
		Region:     regionName,
		Name:       profile.Name,
		VCpu:       irs.VCpuInfo{Count: strconv.Itoa(profile.VCPU), ClockGHz: "-1"},
		MemSizeMiB: strconv.Itoa(profile.Memory * 1024),
		DiskSizeGB: "-1",
	}

	if profile.GPUCount > 0 && profile.GPUModel != "" {
		gpuMfr := "NVIDIA"
		vmSpecInfo.Gpu = []irs.GpuInfo{
			{
				Count:          strconv.Itoa(profile.GPUCount),
				Mfr:            gpuMfr,
				Model:          profile.GPUModel,
				MemSizeGB:      "-1",
				TotalMemSizeGB: "-1",
			},
		}
	} else {
		vmSpecInfo.Gpu = []irs.GpuInfo{}
	}

	vmSpecInfo.KeyValueList = []irs.KeyValue{
		{Key: "Family", Value: profile.Family},
		{Key: "Architecture", Value: profile.Architecture},
		{Key: "Generation", Value: profile.Generation},
	}

	productInfo.VMSpecInfo = vmSpecInfo

	var totalPrice float64 = 0
	var allPriceDetails []map[string]interface{}

	for _, metric := range priceInfo.Metrics {
		if metric.ChargeUnitName == "INSTANCE_HOURS_MULTI_TENANT" ||
			metric.ChargeUnitName == "INSTANCE_HOURS_MULTI_TENANT_TDX" {

			var metricPrice float64 = 0
			var metricCurrency string = "USD"
			var foundUSAPrice bool = false

			for _, amount := range metric.Amounts {
				if amount.Country == "USA" && amount.Currency == "USD" && len(amount.Prices) > 0 {
					metricPrice = amount.Prices[0].Price
					metricCurrency = amount.Currency
					foundUSAPrice = true
					break
				}
			}

			if !foundUSAPrice {
				for _, amount := range metric.Amounts {
					if amount.Currency == "USD" && len(amount.Prices) > 0 {
						metricPrice = amount.Prices[0].Price
						metricCurrency = amount.Currency
						break
					}
				}
			}

			var description string
			if metric.ChargeUnitName == "INSTANCE_HOURS_MULTI_TENANT" {
				description = fmt.Sprintf("Standard Instance-Hours for %s (%d vCPU, %d GB)",
					profile.Name, profile.VCPU, profile.Memory)
			} else if metric.ChargeUnitName == "INSTANCE_HOURS_MULTI_TENANT_TDX" {
				description = fmt.Sprintf("TDX Instance-Hours for %s (%d vCPU, %d GB)",
					profile.Name, profile.VCPU, profile.Memory)
			}

			totalPrice += metricPrice

			priceDetail := map[string]interface{}{
				"metricID":       metric.MetricID,
				"chargeUnitName": metric.ChargeUnitName,
				"price":          metricPrice,
				"currency":       metricCurrency,
				"unit":           metric.ChargeUnit,
				"description":    description,
			}
			allPriceDetails = append(allPriceDetails, priceDetail)
		}
	}

	var onDemand irs.OnDemand

	if len(allPriceDetails) > 0 {
		onDemand = irs.OnDemand{
			PricingId: "TOTAL",
			Unit:      "Hour",
			Currency:  "USD",
			Price:     strconv.FormatFloat(math.Round(totalPrice*1000)/1000, 'f', 3, 64),
			Description: fmt.Sprintf("Total hourly cost for %s (%d vCPU, %d GB)",
				profile.Name, profile.VCPU, profile.Memory),
		}
	}

	matchesFilter := true
	isOnDemandFilterExist, fields := isFieldToFilterExist(irs.OnDemand{}, filterList)

	if isOnDemandFilterExist {
		if !isPicked(onDemand, fields, filterList) {
			matchesFilter = false
		}
	}

	isProductInfoFilterExist, fields := isFieldToFilterExist(irs.ProductInfo{}, filterList)
	if isProductInfoFilterExist && !isPicked(productInfo, fields, filterList) {
		matchesFilter = false
	}

	if !matchesFilter || onDemand.Price == "" {
		return price, fmt.Errorf("Profile %s doesn't match filters or has no pricing information", profile.Name)
	}

	price = irs.Price{
		ProductInfo: productInfo,
		PriceInfo: irs.PriceInfo{
			OnDemand: onDemand,
			CSPPriceInfo: map[string]interface{}{
				"priceInfo":    priceInfo,
				"priceDetails": allPriceDetails,
			},
		},
	}

	return price, nil
}

func containsRegionInGeoTags(geoTags []string, regionName string) bool {
	for _, tag := range geoTags {
		if tag == regionName {
			return true
		}
	}
	return false
}

func isGen2Profile(profileName string) bool {
	if profileName == "gen2-instance" {
		return true
	}

	parts := strings.Split(profileName, "-")
	if len(parts) < 1 {
		return false
	}

	firstPart := parts[0]

	if strings.Contains(firstPart, "x3") || strings.Contains(profileName, "-gen3") {
		return false
	}

	if strings.HasSuffix(firstPart, "x2") ||
		(strings.Contains(firstPart, "x2") && len(firstPart) > 3) {
		return true
	}

	if strings.HasSuffix(firstPart, "z2") ||
		(strings.Contains(firstPart, "z2") && len(firstPart) > 3) {
		return true
	}

	return false
}
