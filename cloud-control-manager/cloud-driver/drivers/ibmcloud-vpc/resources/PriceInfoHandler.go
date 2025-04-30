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
					// Only accept name starts with 'is.' or find kubernetes
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

// VPC Profile structure definition
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

// ServicePlan represents IBM Cloud service plan information
type ServicePlan struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ID          string `json:"id"`
	ChildrenURL string `json:"children_url"`
}

// ProfilePricing represents the pricing information for a specific profile
type ProfilePricing struct {
	ProfileName  string
	RegionName   string
	MetricPrices map[string]struct {
		Price    float64
		Currency string
		Unit     string
	}
}

// ExtendedPricingInfo represents pricing information with metrics
type ExtendedPricingInfo struct {
	Metrics []PriceMetric `json:"metrics"`
	Origin  string        `json:"origin"`
	Type    string        `json:"type"`
}

// ExtendedMetadata represents metadata with extended pricing info
type ExtendedMetadata struct {
	Deployment struct {
		Location string `json:"location"`
	} `json:"deployment"`
	Pricing      ExtendedPricingInfo `json:"pricing"`
	RcCompatible bool                `json:"rc_compatible"`
}

// ExtendedResource represents a resource with extended metadata for pricing
type ExtendedResource struct {
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	ChildrenURL string           `json:"children_url"`
	GeoTags     []string         `json:"geo_tags"`
	Metadata    ExtendedMetadata `json:"metadata"`
	OverviewUI  OverviewUI       `json:"overview_ui"`
}

// ExtendedResourceInfo represents resource info with extended resources
type ExtendedResourceInfo struct {
	Offset        int                `json:"offset"`
	Limit         int                `json:"limit"`
	Count         int                `json:"count"`
	ResourceCount int                `json:"resource_count"`
	First         string             `json:"first"`
	Next          string             `json:"next"`
	Resources     []ExtendedResource `json:"resources"`
}

// GetProfilesForRegion returns VPC profiles for the specified region
func (priceInfoHandler *IbmPriceInfoHandler) GetProfilesForRegion(regionName string) ([]VPCProfile, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "GetProfilesForRegion()")
	start := call.Start()

	// Get profile list using IBM VPC API
	vmSpecHandler := &IbmVmSpecHandler{
		CredentialInfo: priceInfoHandler.CredentialInfo,
		Region:         priceInfoHandler.Region,
		VpcService:     priceInfoHandler.VpcService,
		Ctx:            priceInfoHandler.Ctx,
	}

	// Get VM spec list
	specList, err := vmSpecHandler.ListVMSpec()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to list instance profiles. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	// Transform and filter profile information
	var profiles []VPCProfile
	for _, spec := range specList {
		// Parse vCPU info
		vcpu, err := strconv.Atoi(spec.VCpu.Count)
		if err != nil {
			cblogger.Error(fmt.Sprintf("Failed to parse vCPU count: %s", err))
			vcpu = 1 // Default value
		}

		// Parse memory info (convert MiB to GB)
		memory := 1 // Default value
		if spec.MemSizeMiB != "" && spec.MemSizeMiB != "-1" {
			memMiB, err := strconv.Atoi(spec.MemSizeMiB)
			if err == nil {
				memory = memMiB / 1024 // Convert MiB to GB
				if memory < 1 {
					memory = 1 // Ensure minimum 1GB
				}
			}
		}

		// Extract GPU info
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
					break // Use only first GPU info
				}
			}
		}

		// Determine generation (Gen2, Gen3, etc.)
		generation := "unknown"
		if isGen2Profile(spec.Name) {
			generation = "gen2"
		} else if strings.Contains(spec.Name, "x3") || strings.Contains(spec.Name, "-gen3") {
			generation = "gen3"
		}

		// Extract architecture and family information
		architecture := "amd64" // Default (x86)
		family := ""            // Default family

		for _, kv := range spec.KeyValueList {
			if kv.Key == "Architecture" {
				architecture = kv.Value
			} else if kv.Key == "Family" {
				family = kv.Value
			}
		}

		// Detect s390x architecture from profile name
		if strings.Contains(spec.Name, "z2") && architecture == "amd64" {
			architecture = "s390x" // IBM Z architecture
		}

		// Extract family from name if not provided
		if family == "" {
			if parts := strings.Split(spec.Name, "-"); len(parts) > 0 {
				firstPart := parts[0]
				// First letter indicates family
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

					// Indicate IBM Z architecture if detected
					if strings.Contains(firstPart, "z2") {
						family = family + "_z"
					}
				}
			}
		}

		// Store profile information
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

// GetServicePlans retrieves all service plans for a product family
func (priceInfoHandler *IbmPriceInfoHandler) GetServicePlans(productFamily string) ([]ServicePlan, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "GetServicePlans()")
	start := call.Start()

	// Get initial API response to determine total count
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

	// Retrieve all pages of service plans
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

				// Convert resources to ServicePlan objects
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

// GetPriceInfo retrieves pricing information for a given product family and region
func (priceInfoHandler *IbmPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "GetPriceInfo()")
	start := call.Start()

	// Step 1: Retrieve profiles and service plans
	// Get all profiles for the region
	profiles, err := priceInfoHandler.GetProfilesForRegion(regionName)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get region profiles. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	cblogger.Info(fmt.Sprintf("Retrieved %d profiles for region %s", len(profiles), regionName))

	// Get service plans for the product family
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
		// Step 2: Get Gen2 profile pricing
		// Find the gen2-instance service plan
		var gen2Plan *ServicePlan
		for i, plan := range servicePlans {
			if plan.Name == "gen2-instance" {
				gen2Plan = &servicePlans[i]
				break
			}
		}

		if gen2Plan != nil {
			cblogger.Info(fmt.Sprintf("Found Gen2 service plan: %s", gen2Plan.Name))

			// Filter Gen2 profiles
			var gen2Profiles []VPCProfile
			for _, profile := range profiles {
				if profile.Generation == "gen2" {
					gen2Profiles = append(gen2Profiles, profile)
				}
			}
			cblogger.Info(fmt.Sprintf("Found %d Gen2 profiles to get pricing for", len(gen2Profiles)))

			// Get prices for Gen2 profiles
			gen2ProfilePrices, err := priceInfoHandler.GetGen2ProfilePrices(gen2Plan.ChildrenURL, regionName, gen2Profiles)
			if err != nil {
				cblogger.Error(fmt.Sprintf("Error getting Gen2 profile prices: %s", err.Error()))
				// Continue even with errors to try getting Gen3 prices
			} else {
				// Extract product and price information for Gen2 profiles
				for _, profilePricing := range gen2ProfilePrices {
					// Find matching profile
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

		// Step 3: Get Gen3 profile pricing
		// Filter Gen3 profiles
		var gen3Profiles []VPCProfile
		for _, profile := range profiles {
			if profile.Generation != "gen2" {
				gen3Profiles = append(gen3Profiles, profile)
			}
		}
		cblogger.Info(fmt.Sprintf("Found %d Gen3/other profiles to process", len(gen3Profiles)))

		// Process each Gen3 profile
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

				// Get price for this profile
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

	// Create cloud price data structure
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

	// Marshal to JSON
	data, err := json.Marshal(cloudPriceData)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to marshal price data. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	LoggingInfo(hiscallInfo, start)
	return string(data), nil
}

// GetGen2ProfilePrices retrieves pricing for Gen2 profiles
func (priceInfoHandler *IbmPriceInfoHandler) GetGen2ProfilePrices(childrenURL string, regionName string, gen2Profiles []VPCProfile) ([]ProfilePricing, error) {
	hiscallInfo := GetCallLogScheme(priceInfoHandler.Region, call.PRICEINFO, "PriceInfo", "GetGen2ProfilePrices()")
	start := call.Start()

	// Get initial response to determine total count
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

	// Retrieve all deployment resources
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

	// Filter resources for the target region and map to profiles
	var regionResources []ExtendedResource

	// Standard pattern: "profileName-regionName"
	profileNameRegex := regexp.MustCompile(`^([a-z0-9-]+)-` + regionName + "$")

	// Find region resources
	for _, res := range allResources {
		matched := false

		// Check basic regex pattern (profileName-regionName)
		matches := profileNameRegex.FindStringSubmatch(res.Name)
		if len(matches) > 1 {
			profileName := matches[1]

			// Check if this profile exists in our list
			for _, profile := range gen2Profiles {
				if profile.Name == profileName {
					cblogger.Info(fmt.Sprintf("Found matching resource for profile: %s", profile.Name))
					regionResources = append(regionResources, res)
					matched = true
					break
				}
			}

			if matched {
				continue // Go to next resource
			}
		}

		// Check if profile name is contained within resource name
		for _, profile := range gen2Profiles {
			// Add boundary conditions for more accurate matching
			// E.g., to prevent "bx2" from matching "bx2d"
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
			continue // Go to next resource
		}

		// Check for IBM Z architecture patterns
		for _, profile := range gen2Profiles {
			// Check for z2 pattern (IBM Z architecture)
			if strings.Contains(profile.Name, "z2") {
				// Match Z architecture specific patterns
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
			continue // Go to next resource
		}

		// Region-based matching (fallback method)
		if res.Metadata.Deployment.Location == regionName ||
			containsRegionInGeoTags(res.GeoTags, regionName) {
			// Add as region resource even without explicit profile matching
			cblogger.Info(fmt.Sprintf("Found resource for region by deployment location/geo_tags: %s", res.Name))
			regionResources = append(regionResources, res)
		}
	}

	cblogger.Info(fmt.Sprintf("Found %d resources matching region %s", len(regionResources), regionName))

	// Get detailed pricing information for each profile
	var profilePricings []ProfilePricing
	targetChargeUnits := []string{
		"MEMORY_HOURS",
		"VCPU_HOURS",
		"IS_STORAGE_GIGABYTE_HOURS",
		"IS_LARGE_STORAGE_GIGABYTE_HOURS",
		"V100_HOURS",
		"SECUREEXECUTION_VCPU_HOURS",
		"SECUREEXECUTION_MEMORY_HOURS",
		"Z_VCPU_HOURS",             // IBM Z vCPU Hours
		"Z_MEMORY_HOURS",           // IBM Z Memory Hours
		"Z_STORAGE_GIGABYTE_HOURS", // IBM Z Storage Hours
	}

	// Create profile name pattern mapping
	profilePatterns := make(map[string][]string)

	// Generate matching patterns for each profile
	for _, profile := range gen2Profiles {
		patterns := []string{
			profile.Name,                    // Exact name matching
			profile.Name + "-" + regionName, // name-region pattern
		}

		// Support for x86 extended profiles (bx2/bx2d)
		if strings.Contains(profile.Name, "x2") {
			baseName := strings.Replace(profile.Name, "d", "", -1) // Remove 'd' (bx2d -> bx2)
			if baseName != profile.Name {
				patterns = append(patterns, baseName) // Add base name to patterns
			}
		}

		// Support for Z architecture profiles (bz2/bz2e)
		if strings.Contains(profile.Name, "z2") {
			baseName := strings.Replace(profile.Name, "e", "", -1) // Remove 'e' (bz2e -> bz2)
			if baseName != profile.Name {
				patterns = append(patterns, baseName) // Add base name to patterns
			}

			// Z architecture related keywords
			patterns = append(patterns, "ibmz", "s390x", "z-arch")
		}

		profilePatterns[profile.Name] = patterns
	}

	// Extract pricing information from resources
	for _, res := range regionResources {
		// Extract profile name from resource
		matchedProfileName := ""
		highestMatchScore := 0

		// Calculate matching score for all profiles
		for profileName, patterns := range profilePatterns {
			for _, pattern := range patterns {
				if strings.Contains(res.Name, pattern) {
					// Longer pattern means more accurate matching
					matchScore := len(pattern)
					if matchScore > highestMatchScore {
						highestMatchScore = matchScore
						matchedProfileName = profileName
					}
				}
			}
		}

		// If no profile match found, try regex matching
		if matchedProfileName == "" {
			matches := profileNameRegex.FindStringSubmatch(res.Name)
			if len(matches) > 1 {
				baseProfileName := matches[1]

				// Check if this base profile name matches any of our profiles
				for _, profile := range gen2Profiles {
					if strings.HasPrefix(profile.Name, baseProfileName) ||
						strings.HasPrefix(baseProfileName, profile.Name) {
						matchedProfileName = profile.Name
						break
					}
				}
			}

			// Still no match, log and continue
			if matchedProfileName == "" {
				cblogger.Warning(fmt.Sprintf("Could not determine profile for resource: %s", res.Name))
				continue
			}
		}

		cblogger.Info(fmt.Sprintf("Processing pricing for profile: %s from resource: %s",
			matchedProfileName, res.Name))

		// Extract pricing from metadata
		if len(res.Metadata.Pricing.Metrics) == 0 {
			cblogger.Warning(fmt.Sprintf("Resource %s has no pricing metrics", res.Name))
			continue
		}

		// Extract pricing for specific charge unit types
		metricPrices := make(map[string]struct {
			Price    float64
			Currency string
			Unit     string
		})

		for _, metric := range res.Metadata.Pricing.Metrics {
			chargeUnitName := metric.ChargeUnitName

			// Check if this is one of our target charge units
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

			// Find USA/USD price or fallback to other pricing
			var bestPrice struct {
				Price    float64
				Currency string
			}

			// First try to find USA/USD price
			hasUSAPrice := false
			for _, amount := range metric.Amounts {
				if amount.Country == "USA" && amount.Currency == "USD" && len(amount.Prices) > 0 {
					bestPrice.Price = amount.Prices[0].Price
					bestPrice.Currency = amount.Currency
					hasUSAPrice = true
					break
				}
			}

			// If no USA price, use other price
			if !hasUSAPrice && len(metric.Amounts) > 0 && len(metric.Amounts[0].Prices) > 0 {
				bestPrice.Price = metric.Amounts[0].Prices[0].Price
				bestPrice.Currency = metric.Amounts[0].Currency
			}

			// Store price information
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

		// Add pricing to profile
		if len(metricPrices) > 0 {
			// Find existing profile pricing (to avoid duplicates)
			existingIndex := -1
			for i, pp := range profilePricings {
				if pp.ProfileName == matchedProfileName {
					existingIndex = i
					break
				}
			}

			if existingIndex >= 0 {
				// Merge metricPrices with existing profile
				for k, v := range metricPrices {
					profilePricings[existingIndex].MetricPrices[k] = v
				}
				cblogger.Info(fmt.Sprintf("Updated pricing for profile %s with %d additional metrics",
					matchedProfileName, len(metricPrices)))
			} else {
				// Add new ProfilePricing
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

	// Special handling: Copy pricing for missing IBM Z profiles
	// Some Z profiles (bz2e, cz2e, mz2e) may have same pricing as base profiles (bz2, cz2, mz2)
	missingZProfiles := make(map[string]string) // key: missing profile, value: base profile

	for _, profile := range gen2Profiles {
		// Check for Z extended profiles (bz2e, cz2e, mz2e)
		if strings.Contains(profile.Name, "z2e") {
			found := false
			for _, pp := range profilePricings {
				if pp.ProfileName == profile.Name {
					found = true
					break
				}
			}

			if !found {
				// Map missing extended profile to base profile
				baseProfileName := strings.Replace(profile.Name, "e", "", -1) // Remove 'e' (bz2e -> bz2)
				missingZProfiles[profile.Name] = baseProfileName
			}
		}
	}

	// Clone pricing for missing Z profiles
	for missingProfile, baseProfile := range missingZProfiles {
		for _, pp := range profilePricings {
			if pp.ProfileName == baseProfile {
				// Clone base profile pricing for missing profile
				newPricing := ProfilePricing{
					ProfileName: missingProfile,
					RegionName:  regionName,
					MetricPrices: make(map[string]struct {
						Price    float64
						Currency string
						Unit     string
					}),
				}

				// Copy metric prices
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

// CreateGen2ProfilePrice creates a price entry for a Gen2 profile
func (priceInfoHandler *IbmPriceInfoHandler) CreateGen2ProfilePrice(profile VPCProfile, profilePricing ProfilePricing, regionName string, filterList []irs.KeyValue) (irs.Price, error) {
	// Check for IBM Z architecture
	isIBMZ := profile.Architecture == "s390x" || strings.Contains(profile.Name, "z2")

	// Create profile description
	var description string
	if isIBMZ {
		description = fmt.Sprintf("IBM VPC Gen2 IBM Z instance profile %s with %d vCPU and %d GB RAM",
			profile.Name, profile.VCPU, profile.Memory)
	} else {
		description = fmt.Sprintf("IBM VPC Gen2 instance profile %s with %d vCPU and %d GB RAM",
			profile.Name, profile.VCPU, profile.Memory)
	}

	// Create product info
	productInfo := irs.ProductInfo{
		ProductId:      fmt.Sprintf("%s-%s", profile.Name, regionName),
		RegionName:     regionName,
		Description:    description,
		CSPProductInfo: profile,
	}

	// Set VM spec info
	vmSpecInfo := irs.VMSpecInfo{
		Region:     regionName,
		Name:       profile.Name,
		VCpu:       irs.VCpuInfo{Count: strconv.Itoa(profile.VCPU), ClockGHz: "-1"},
		MemSizeMiB: strconv.Itoa(profile.Memory * 1024), // GB to MiB
		DiskSizeGB: "-1",
	}

	// Set GPU info if available
	if profile.GPUCount > 0 && profile.GPUModel != "" {
		gpuMfr := "NVIDIA" // Default GPU manufacturer
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

	// Set key-value list with additional architecture info
	vmSpecInfo.KeyValueList = []irs.KeyValue{
		{Key: "Family", Value: profile.Family},
		{Key: "Architecture", Value: profile.Architecture},
		{Key: "Generation", Value: profile.Generation},
	}

	// Add IBM Z architecture specific info
	if isIBMZ {
		vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList,
			irs.KeyValue{Key: "Platform", Value: "IBM Z"})

		// Check for extended profile (bz2e, cz2e, mz2e)
		if strings.HasSuffix(profile.Name, "e") {
			vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList,
				irs.KeyValue{Key: "Secure", Value: "Enhanced"})
		}
	}

	productInfo.VMSpecInfo = vmSpecInfo
	productInfo.OSDistribution = "NA"
	productInfo.PreInstalledSw = "NA"

	// Calculate total price from all charge unit types
	var allPricingPolicies []irs.PricingPolicies
	var totalPrice float64

	// Add all relevant charge types from the pricing data
	for chargeUnitName, priceData := range profilePricing.MetricPrices {
		var price float64
		var description string

		// Calculate specific price based on profile resources
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
			// For other charge types, use base price
			price = priceData.Price
			description = chargeUnitName
		}

		// Add to total price
		totalPrice += price

		// Create pricing policy (for internal use only)
		allPricingPolicies = append(allPricingPolicies, irs.PricingPolicies{
			PricingId:     chargeUnitName,
			PricingPolicy: "OnDemand",
			Unit:          priceData.Unit,
			Currency:      priceData.Currency,
			Price:         strconv.FormatFloat(price, 'f', -1, 64),
			Description:   description,
		})
	}

	// Create the final pricing policies list (only TOTAL)
	var pricingPolicies []irs.PricingPolicies

	// Add total price policy if there are any prices
	if len(allPricingPolicies) > 0 {
		var totalDescription string
		if isIBMZ {
			totalDescription = fmt.Sprintf("Total hourly cost for IBM Z profile %s (%d vCPU, %d GB)",
				profile.Name, profile.VCPU, profile.Memory)
		} else {
			totalDescription = fmt.Sprintf("Total hourly cost for %s (%d vCPU, %d GB)",
				profile.Name, profile.VCPU, profile.Memory)
		}

		pricingPolicies = append(pricingPolicies, irs.PricingPolicies{
			PricingId:     "TOTAL",
			PricingPolicy: "OnDemand",
			Unit:          "Hour",
			Currency:      "USD",
			Price:         strconv.FormatFloat(math.Round(totalPrice*1000)/1000, 'f', 3, 64),
			Description:   totalDescription,
		})
	}

	// Apply filtering (only for TOTAL since that's all we're returning)
	matchesFilter := true
	isPricingPoliciesFilterExist, fields := isFieldToFilterExist(irs.PricingPolicies{}, filterList)
	if isPricingPoliciesFilterExist && len(pricingPolicies) > 0 {
		matchesFilter = false
		for _, policy := range pricingPolicies {
			if isPicked(policy, fields, filterList) {
				matchesFilter = true
				break
			}
		}
	}

	isProductInfoFilterExist, fields := isFieldToFilterExist(irs.ProductInfo{}, filterList)
	if isProductInfoFilterExist && !isPicked(productInfo, fields, filterList) {
		matchesFilter = false
	}

	if !matchesFilter || len(pricingPolicies) == 0 {
		return irs.Price{}, fmt.Errorf("Profile %s doesn't match filters or has no pricing policies", profile.Name)
	}

	return irs.Price{
		ProductInfo: productInfo,
		PriceInfo: irs.PriceInfo{
			PricingPolicies: pricingPolicies,
			CSPPriceInfo:    profilePricing,
		},
	}, nil
}

// GetGen3ProfilePrice retrieves pricing for a Gen3 or other profile
func (priceInfoHandler *IbmPriceInfoHandler) GetGen3ProfilePrice(childrenURL string, profile VPCProfile, regionName string, filterList []irs.KeyValue) (irs.Price, error) {
	var price irs.Price

	// Get initial response to determine total count
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

	// Retrieve all resources
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

	// Find the resource for the target region
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

	// Get pricing information
	response, err := getIbmPriceInfo(pricingURL)
	if err != nil {
		return price, err
	}

	var priceInfo PriceInfo
	err = json.Unmarshal(response, &priceInfo)
	if err != nil {
		return price, err
	}

	// Create product info
	productInfo := irs.ProductInfo{
		ProductId:      fmt.Sprintf("%s-%s", profile.Name, regionName), // {profile name}-{region} format
		RegionName:     regionName,
		Description:    fmt.Sprintf("IBM VPC %s instance profile %s with %d vCPU and %d GB RAM", profile.Generation, profile.Name, profile.VCPU, profile.Memory),
		CSPProductInfo: profile,
	}

	// Set VM spec info
	vmSpecInfo := irs.VMSpecInfo{
		Region:     regionName,
		Name:       profile.Name,
		VCpu:       irs.VCpuInfo{Count: strconv.Itoa(profile.VCPU), ClockGHz: "-1"},
		MemSizeMiB: strconv.Itoa(profile.Memory * 1024), // GB to MiB
		DiskSizeGB: "-1",
	}

	// Set GPU info if available
	if profile.GPUCount > 0 && profile.GPUModel != "" {
		gpuMfr := "NVIDIA" // Default GPU manufacturer
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

	// Set key-value list
	vmSpecInfo.KeyValueList = []irs.KeyValue{
		{Key: "Family", Value: profile.Family},
		{Key: "Architecture", Value: profile.Architecture},
		{Key: "Generation", Value: profile.Generation},
	}

	productInfo.VMSpecInfo = vmSpecInfo
	productInfo.OSDistribution = "NA"
	productInfo.PreInstalledSw = "NA"

	// Find pricing information for INSTANCE_HOURS_MULTI_TENANT and INSTANCE_HOURS_MULTI_TENANT_TDX
	var allPricingPolicies []irs.PricingPolicies
	var totalPrice float64 = 0

	for _, metric := range priceInfo.Metrics {
		// only consider INSTANCE_HOURS_MULTI_TENANT and INSTANCE_HOURS_MULTI_TENANT_TDX
		if metric.ChargeUnitName == "INSTANCE_HOURS_MULTI_TENANT" ||
			metric.ChargeUnitName == "INSTANCE_HOURS_MULTI_TENANT_TDX" {

			var metricPrice float64 = 0
			var metricCurrency string = "USD"
			var foundUSAPrice bool = false

			// 우선 USA/USD 가격을 찾음
			for _, amount := range metric.Amounts {
				if amount.Country == "USA" && amount.Currency == "USD" && len(amount.Prices) > 0 {
					metricPrice = amount.Prices[0].Price
					metricCurrency = amount.Currency
					foundUSAPrice = true
					break
				}
			}

			// if no USA price found, fallback to other prices
			if !foundUSAPrice {
				for _, amount := range metric.Amounts {
					if amount.Currency == "USD" && len(amount.Prices) > 0 {
						metricPrice = amount.Prices[0].Price
						metricCurrency = amount.Currency
						break
					}
				}
			}

			// generate description based on charge unit name
			var description string
			if metric.ChargeUnitName == "INSTANCE_HOURS_MULTI_TENANT" {
				description = fmt.Sprintf("Standard Instance-Hours for %s (%d vCPU, %d GB)",
					profile.Name, profile.VCPU, profile.Memory)
			} else if metric.ChargeUnitName == "INSTANCE_HOURS_MULTI_TENANT_TDX" {
				description = fmt.Sprintf("TDX Instance-Hours for %s (%d vCPU, %d GB)",
					profile.Name, profile.VCPU, profile.Memory)
			}

			totalPrice += metricPrice

			allPricingPolicies = append(allPricingPolicies, irs.PricingPolicies{
				PricingId:     metric.MetricID,
				PricingPolicy: "OnDemand",
				Unit:          metric.ChargeUnit,
				Currency:      metricCurrency,
				Price:         strconv.FormatFloat(math.Round(metricPrice*1000)/1000, 'f', 3, 64),
				Description:   description,
			})
		}
	}

	// Create final pricing policies list (only TOTAL)
	var pricingPolicies []irs.PricingPolicies

	if len(allPricingPolicies) > 0 {
		pricingPolicies = append(pricingPolicies, irs.PricingPolicies{
			PricingId:     "TOTAL",
			PricingPolicy: "OnDemand",
			Unit:          "Hour",
			Currency:      "USD",
			Price:         strconv.FormatFloat(math.Round(totalPrice*1000)/1000, 'f', 3, 64),
			Description: fmt.Sprintf("Total hourly cost for %s (%d vCPU, %d GB)",
				profile.Name, profile.VCPU, profile.Memory),
		})
	}

	// Apply filtering (only for TOTAL since that's all we're returning)
	matchesFilter := true
	isPricingPoliciesFilterExist, fields := isFieldToFilterExist(irs.PricingPolicies{}, filterList)
	if isPricingPoliciesFilterExist {
		matchesFilter = false
		for _, policy := range pricingPolicies {
			if isPicked(policy, fields, filterList) {
				matchesFilter = true
				break
			}
		}
	}

	isProductInfoFilterExist, fields := isFieldToFilterExist(irs.ProductInfo{}, filterList)
	if isProductInfoFilterExist && !isPicked(productInfo, fields, filterList) {
		matchesFilter = false
	}

	if !matchesFilter || len(pricingPolicies) == 0 {
		return price, fmt.Errorf("Profile %s doesn't match filters or has no pricing policies", profile.Name)
	}

	price = irs.Price{
		ProductInfo: productInfo,
		PriceInfo: irs.PriceInfo{
			PricingPolicies: pricingPolicies, // Only return the TOTAL pricing policy
			CSPPriceInfo:    priceInfo,       // Keep original PriceInfo
		},
	}

	return price, nil
}

// Helper function to check if a region is in geo_tags
func containsRegionInGeoTags(geoTags []string, regionName string) bool {
	for _, tag := range geoTags {
		if tag == regionName {
			return true
		}
	}
	return false
}

// Helper function to check if a profile is Gen2
func isGen2Profile(profileName string) bool {
	// "gen2-instance" is explicitly Gen2
	if profileName == "gen2-instance" {
		return true
	}

	// Split the profile name to analyze parts
	parts := strings.Split(profileName, "-")
	if len(parts) < 1 {
		return false
	}

	firstPart := parts[0]

	// Gen3 profiles are explicitly excluded - check only first part
	if strings.Contains(firstPart, "x3") || strings.Contains(profileName, "-gen3") {
		return false
	}

	// Check x86 architecture Gen2 patterns
	// - Base x86 patterns: ends with "x2" (e.g., bx2, cx2, mx2, gx2, ox2)
	// - Extended x86 patterns: contains "x2" and has additional suffix (e.g., bx2d, cx2d, mx2d)
	if strings.HasSuffix(firstPart, "x2") ||
		(strings.Contains(firstPart, "x2") && len(firstPart) > 3) {
		return true
	}

	// Check IBM Z architecture Gen2 patterns
	// - Base Z patterns: ends with "z2" (e.g., bz2, cz2, mz2)
	// - Extended Z patterns: contains "z2" and has additional suffix (e.g., bz2e, cz2e, mz2e)
	if strings.HasSuffix(firstPart, "z2") ||
		(strings.Contains(firstPart, "z2") && len(firstPart) > 3) {
		return true
	}

	return false
}
