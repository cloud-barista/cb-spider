package resources

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"google.golang.org/api/cloudbilling/v1"
	cbb "google.golang.org/api/cloudbilling/v1beta"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var validFilterKey map[string]bool

const pricingURL = "https://cloud.google.com/compute/all-pricing?hl=en"

type PriceData struct {
	Region      string
	MachineType string
	Amount      float64
	Currency    string
	Unit        string
	Source      string
}

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

// Regex patterns for parsing Google Cloud pricing page
var (
	pOpen  = `(?:\\u003cp\\u003e|<p>)`
	pClose = `(?:\\u003c/p\\u003e|</p>)`

	reMachine = regexp.MustCompile(
		pOpen +
			`([a-z0-9][a-z0-9-]*(?:-[a-z0-9-]+){1,3})` +
			`(?:\s*(?:\\u003ca[^>]*\\u003e.*?\\u003c/a\\u003e|<a[^>]*>.*?</a>))?` +
			pClose,
	)

	rePriceHour = regexp.MustCompile(`\$([0-9][0-9.]*)\s*/\s*1\s*hour`)
	rePriceDash = regexp.MustCompile(pOpen + `\s*-\s*` + pClose)
	reRegion    = regexp.MustCompile(`"([A-Za-z][^"]+?\s?\([a-z0-9-]+\))"`)
)

// fetchPricingData fetches and parses Google Cloud pricing data
func fetchPricingData() (map[string]*PriceData, error) {
	const minDataSize = 100 * 1024
	const maxRetries = 3
	const timeout = 5 * time.Minute

	for attempt := 1; attempt <= maxRetries; attempt++ {
		cblogger.Infof("Attempt %d/%d: Fetching pricing data from %s", attempt, maxRetries, pricingURL)

		client := &http.Client{Timeout: timeout}

		req, err := http.NewRequest("GET", pricingURL, nil)
		if err != nil {
			cblogger.Error("Failed to create request:", err)
			continue
		}

		// Set headers
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")

		resp, err := client.Do(req)
		if err != nil {
			cblogger.Error("HTTP request failed:", err)
			if attempt < maxRetries {
				time.Sleep(5 * time.Second)
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			cblogger.Errorf("HTTP request returned status: %d", resp.StatusCode)
			if attempt < maxRetries {
				time.Sleep(5 * time.Second)
			}
			continue
		}

		// Read response body
		data := make([]byte, 0)
		buffer := make([]byte, 32*1024)

		var reader = resp.Body
		if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(resp.Body)
			if err != nil {
				cblogger.Error("Failed to create gzip reader:", err)
				continue
			}
			defer gzReader.Close()
			reader = gzReader
		}

		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				data = append(data, buffer[:n]...)
			}
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				cblogger.Error("Failed to read response body:", err)
				break
			}
		}

		if len(data) < minDataSize {
			cblogger.Errorf("Data size (%d bytes) is less than minimum required (%d bytes)", len(data), minDataSize)
			if attempt < maxRetries {
				time.Sleep(5 * time.Second)
			}
			continue
		}

		cblogger.Infof("Successfully fetched %d bytes", len(data))

		// Parse the data and return price cache
		priceCache, err := parsePricingData(data)
		if err != nil {
			cblogger.Error("Failed to parse pricing data:", err)
			if attempt < maxRetries {
				time.Sleep(5 * time.Second)
			}
			continue
		}

		return priceCache, nil
	}

	return nil, errors.New("failed to fetch pricing data after all retry attempts")
}

// findBlocksByBrackets finds pricing blocks in the HTML
func findBlocksByBrackets(b []byte) [][2]int {
	s := string(b)
	var spans [][2]int
	i := 0
	for {
		e := strings.Index(s[i:], "[3]]")
		if e == -1 {
			break
		}
		e += i
		st := strings.LastIndex(s[:e], "[[[[")
		if st != -1 {
			spans = append(spans, [2]int{st, e + len("[3]]")})
		}
		i = e + len("[3]]")
	}
	return spans
}

// parseBlock parses a pricing block and extracts price data
func parseBlock(block []byte, sectionIdx int) []*PriceData {
	s := string(block)
	s = strings.ReplaceAll(s, "\u00a0", " ")

	region := ""
	if m := reRegion.FindAllStringSubmatch(s, -1); len(m) > 0 {
		region = m[len(m)-1][1]
	}

	if region == "" {
		return []*PriceData{}
	}

	matches := reMachine.FindAllStringSubmatchIndex(s, -1)

	var out []*PriceData
	for _, loc := range matches {
		machineType := s[loc[2]:loc[3]]

		specStart := loc[1]
		specEnd := specStart + 1200
		if specEnd > len(s) {
			specEnd = len(s)
		}
		frag := s[specStart:specEnd]

		if rePriceDash.MatchString(frag) {
			continue
		}

		ploc := rePriceHour.FindStringSubmatchIndex(frag)
		if ploc == nil {
			continue
		}

		amt, err := strconv.ParseFloat(frag[ploc[2]:ploc[3]], 64)
		if err != nil {
			continue
		}

		out = append(out, &PriceData{
			Region:      region,
			MachineType: machineType,
			Amount:      amt,
			Currency:    "USD",
			Unit:        "Hour",
			Source:      pricingURL,
		})
	}

	return out
}

// coalesceByRegionMachine removes duplicates, keeping the highest price
func coalesceByRegionMachine(priceData []*PriceData) []*PriceData {
	type key struct{ reg, m string }
	best := make(map[key]*PriceData)

	for _, data := range priceData {
		k := key{reg: data.Region, m: data.MachineType}
		if prev, ok := best[k]; ok {
			if data.Amount > prev.Amount {
				best[k] = data
			}
		} else {
			best[k] = data
		}
	}

	out := make([]*PriceData, 0, len(best))
	for _, v := range best {
		out = append(out, v)
	}
	return out
}

// parsePricingData parses the fetched HTML data and returns the price cache
func parsePricingData(data []byte) (map[string]*PriceData, error) {
	spans := findBlocksByBrackets(data)
	if len(spans) == 0 {
		return nil, errors.New("no pricing blocks found")
	}

	var allPriceData []*PriceData
	for i, sp := range spans {
		blk := data[sp[0]:sp[1]]
		blockPriceData := parseBlock(blk, i)
		allPriceData = append(allPriceData, blockPriceData...)
	}

	allPriceData = coalesceByRegionMachine(allPriceData)
	if len(allPriceData) == 0 {
		return nil, errors.New("no valid pricing data found")
	}

	// Create price cache
	priceCache := make(map[string]*PriceData)

	// Populate cache with new data
	for _, priceData := range allPriceData {
		key := fmt.Sprintf("%s:%s", priceData.Region, priceData.MachineType)
		priceCache[key] = priceData
	}

	cblogger.Infof("Created pricing cache with %d entries", len(priceCache))

	return priceCache, nil
}

// getPriceFromCache retrieves price data from cache
func getPriceFromCache(priceCache map[string]*PriceData, region, machineType string) *PriceData {
	key := fmt.Sprintf("%s:%s", region, machineType)
	return priceCache[key]
}

func (priceInfoHandler *GCPPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, additionalFilterList []irs.KeyValue) (string, error) {
	priceLists := make([]irs.Price, 0)

	filter, isValid := filterListToMap(additionalFilterList)

	cblogger.Infof(">>> filter key is %v\n", isValid)
	if !isValid {
		return "", errors.New("invalid filter key")
	}

	// Fetch pricing data directly each time
	priceCache, err := fetchPricingData()
	if err != nil {
		cblogger.Error("Failed to fetch pricing data:", err)
		return "", err
	}

	cblogger.Infof("filter value : %+v", additionalFilterList)

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

					// Get price from fetched data instead of cache
					priceData := getPriceFromCacheForRegion(priceCache, regionName, machineType.Name)
					if priceData == nil {
						cblogger.Infof("No pricing data found for machine type %s in region %s, skipping", machineType.Name, regionName)
						continue
					}

					priceInfo, err := mappingToPriceInfoFromCache(priceData, filter)
					if err != nil {
						cblogger.Error("error occurred while mapping the pricing info struct; machine type:", machineType.Name, ", message:", err)
						return "", err
					}

					cblogger.Infof("fetch :: %s machine type with price $%f", productInfo.VMSpecInfo.Name, priceData.Amount)

					aPrice := irs.Price{
						ZoneName:    machineType.Zone,
						ProductInfo: *productInfo,
						PriceInfo:   *priceInfo,
					}

					priceLists = append(priceLists, aPrice)
				}
			}
		}
	}

	cloudPrice := irs.CloudPrice{
		Meta:       irs.Meta{Version: "0.5", Description: "Multi-Cloud Price Info"},
		CloudName:  "GCP",
		RegionName: regionName,
		PriceList:  priceLists,
	}

	convertedPriceData, err := ConvertJsonStringNoEscape(cloudPrice)
	if err != nil {
		cblogger.Error("error occurred when removing escape characters from the response struct;", err)
		return "", err
	}

	return convertedPriceData, nil
}

// getPriceFromCacheForRegion gets price data from cache, trying to match region names
func getPriceFromCacheForRegion(priceCache map[string]*PriceData, regionName, machineType string) *PriceData {
	// Try exact match first
	priceData := getPriceFromCache(priceCache, regionName, machineType)
	if priceData != nil {
		return priceData
	}

	// Try to find a matching region in cache (Google's pricing page may have different region naming)
	for _, data := range priceCache {
		if data.MachineType == machineType {
			// Check if region names are similar (contains the regionName or vice versa)
			if strings.Contains(strings.ToLower(data.Region), strings.ToLower(regionName)) ||
				strings.Contains(strings.ToLower(regionName), strings.ToLower(data.Region)) {
				return data
			}
		}
	}

	return nil
}

// mappingToPriceInfoFromCache creates PriceInfo from cached price data
func mappingToPriceInfoFromCache(priceData *PriceData, filter map[string]*string) (*irs.PriceInfo, error) {
	onDemand := irs.OnDemand{
		PricingId:   "NA",
		Unit:        priceData.Unit,
		Currency:    priceData.Currency,
		Price:       fmt.Sprintf("%g", priceData.Amount),
		Description: fmt.Sprintf("OnDemand pricing for %s", priceData.MachineType),
	}

	return &irs.PriceInfo{
		OnDemand:     onDemand,
		CSPPriceInfo: priceData,
	}, nil
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

	gpuInfoList := []irs.GpuInfo{}
	if machineType.Accelerators != nil {
		gpuInfoList = acceleratorsToGPUInfoList(machineType.Accelerators)
	}

	productInfo.VMSpecInfo.Name = machineType.Name
	productInfo.VMSpecInfo.VCpu.Count = fmt.Sprintf("%d", machineType.GuestCpus)
	productInfo.VMSpecInfo.VCpu.ClockGHz = "-1"
	productInfo.VMSpecInfo.MemSizeMiB = fmt.Sprintf("%d", machineType.MemoryMb)
	productInfo.VMSpecInfo.DiskSizeGB = "-1"
	productInfo.VMSpecInfo.Gpu = gpuInfoList

	productInfo.Description = machineType.Description

	return productInfo, nil
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
