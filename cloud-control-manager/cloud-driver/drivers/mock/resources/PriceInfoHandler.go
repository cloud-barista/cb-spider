// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2023.12.

package resources

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type MockPriceInfoHandler struct {
	Region   idrv.RegionInfo
	MockName string
}

const (
	COMPUTE_INSTANCE      = "Compute Instance"
	STORAGE               = "Storage"
	NETWORK_LOAD_BALANCER = "Network Load Balancer"
)

// ====================================================
// ------- vm instance struct for price info
type InstanceData struct {
	Category     string       `json:"category"`
	InstanceName string       `json:"instanceName"`
	InstanceInfo InstanceInfo `json:"instanceInfo"`
	PricingList  PricingList  `json:"pricingList"`
}
type InstanceInfo struct {
	RegionName            string `json:"regionName"`
	InstanceType          string `json:"instanceType"`
	Vcpu                  string `json:"vcpu"`
	Clock                 string `json:"clock"`
	Memory                string `json:"memory"`
	Storage               string `json:"storage"`
	ProcessorArchitecture string `json:"processorArchitecture"`
	Os                    string `json:"os"`
	ProcessorFeatures     string `json:"processorFeatures"`
}

//------- vm instance struct for price info

// ------- storage struct for price info
type StorageData struct {
	Category    string      `json:"category"`
	StorageName string      `json:"storageName"`
	StorageInfo StorageInfo `json:"storageInfo"`
	PricingList PricingList `json:"pricingList"`
}
type StorageInfo struct {
	RegionName  string `json:"regionName"`
	StorageType string `json:"storageType"`
	MaxVolume   string `json:"maxVolume"`
	MaxIOPS     string `json:"maxIOPS"`
}

//------- storage struct for price info

// ------- load balancer struct for price info
type NLBData struct {
	Category    string      `json:"category"`
	NLBName     string      `json:"nlbName"`
	NLBInfo     NLBInfo     `json:"nlbInfo"`
	PricingList PricingList `json:"pricingList"`
}
type NLBInfo struct {
	RegionName string `json:"regionName"`
}

//------- load balancer struct for price info

// ------- common struct for price info
type PricingList struct {
	PayAsYouGo PayAsYouGo   `json:"payAsYouGo"`
	SavingPlan []SavingPlan `json:"savingPlan"`
}
type PayAsYouGo struct {
	PricingId string `json:"priceId"`
	Unit      string `json:"unit"`
	Currency  string `json:"currency"`
	Price     string `json:"price"`
}
type SavingPlan struct {
	PricingId string `json:"priceId"`
	Term      string `json:"term"`
	Unit      string `json:"unit"`
	Currency  string `json:"currency"`
	Price     string `json:"price"`
}

//------- common struct for price info
//====================================================

func (handler *MockPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListProductFamily()!")

	productFamily := []string{
		COMPUTE_INSTANCE,
		STORAGE,
		NETWORK_LOAD_BALANCER,
	}

	return productFamily, nil
}

// 1. Get the Mock's price info from product price info files
//   - by getMockPriceInfo()
//
// 2. transform csp price info to Spider price info(Global view) with filter processing
//   - by transformPriceInfo()
//     -> transformToProductInfo() -> checkFilters()
//     -> transformToPriceInfo()   -> checkFilters()
//     -> make cloudPrice.PriceList info
//     -> make global json for result string
//
// 3. return Spider price info
func (handler *MockPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetPriceInfo()!")

	// if filterList is empty slice, set to nil to make simple processing
	if len(filterList) == 0 {
		filterList = nil
	}

	var data []*interface{}
	switch productFamily {
	case COMPUTE_INSTANCE, STORAGE, NETWORK_LOAD_BALANCER:
		var err error
		data, err = getMockPriceInfo(productFamily, regionName)
		if err != nil {
			cblogger.Error(err)
			return "", err
		}
	default:
		err := errors.New(productFamily + " is not supported product family!")
		cblogger.Error(err)
		return "", err
	}

	resultJson, err := transformPriceInfo(productFamily, data, filterList)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return resultJson, nil
}

func transformPriceInfo(productFamily string, jsonData []*interface{}, filterList []irs.KeyValue) (string, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called transformVMPriceInfo()!")

	var gPriceInfo irs.CloudPriceData
	var cloudPrice irs.CloudPrice

	cloudPrice.CloudName = "Mock"

	for _, v := range jsonData {
		// transform csp price info to Spider price info(Global view)
		hasProductKey, gProductInfo, err := transformToProductInfo(productFamily, v, filterList)
		if err != nil {
			cblogger.Error(err)
			return "", err
		}

		hasPolicyKey, gOnePriceInfo, err := transformToPriceInfo(productFamily, v, filterList)
		if err != nil {
			cblogger.Error(err)
			return "", err
		}

		// if filterList has any filter, but product or policies has no relational keys, then skip
		if filterList != nil && !hasProductKey && !hasPolicyKey {
			continue
		}

		// Add to PriceList only if the both return values are not nil
		if gProductInfo != nil && gOnePriceInfo != nil {
			cloudPrice.PriceList = append(cloudPrice.PriceList, irs.Price{
				ProductInfo: *gProductInfo,
				PriceInfo:   *gOnePriceInfo,
			})
		}
	}

	if len(cloudPrice.PriceList) > 0 {
		gPriceInfo.CloudPriceList = append(gPriceInfo.CloudPriceList, cloudPrice)
	}

	// if gPriceInfo.CloudPriceList is  nil, initialize it to print out as '[]'
	if gPriceInfo.CloudPriceList == nil {
		gPriceInfo.CloudPriceList = []irs.CloudPrice{}
	}

	gPriceInfo.Meta = irs.Meta{
		Version:     "v0.1",
		Description: "Multi-Cloud Price Info",
	}

	globalJsonData, err := json.MarshalIndent(gPriceInfo, "", "    ")
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return string(globalJsonData), nil
}

// transform Mock's product info into global view's productInfo
// jsonData: mock any family's price info
// gProductInfoTemplate: global view's productInfo template
// return: global view's productInfo
func transformToProductInfo(productFamily string, jsonData *interface{}, filterList []irs.KeyValue) (bool /*hasKey*/, *irs.ProductInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called transformProductInfo()!")

	var productInfo irs.ProductInfo

	switch productFamily {
	case COMPUTE_INSTANCE:
		json := (*jsonData).(InstanceData)
		productInfo.ProductId = json.InstanceName
		productInfo.RegionName = json.InstanceInfo.RegionName
		productInfo.RegionName = json.InstanceInfo.RegionName
		productInfo.ZoneName = "NA"
		productInfo.InstanceType = json.InstanceInfo.InstanceType
		productInfo.Vcpu = json.InstanceInfo.Vcpu
		productInfo.Memory = json.InstanceInfo.Memory
		productInfo.Storage = json.InstanceInfo.Storage
		productInfo.Gpu = "NA"
		productInfo.GpuMemory = "NA"
		productInfo.OperatingSystem = json.InstanceInfo.Os
		productInfo.PreInstalledSw = "NA"
		productInfo.Description = json.InstanceInfo.ProcessorArchitecture + ", " +
			json.InstanceInfo.ProcessorFeatures

	case STORAGE:
		json := (*jsonData).(StorageData)
		productInfo.ProductId = json.StorageName
		productInfo.RegionName = json.StorageInfo.RegionName
		productInfo.ZoneName = "NA"
		productInfo.VolumeType = json.StorageInfo.StorageType
		productInfo.StorageMedia = "NA"
		productInfo.MaxVolumeSize = json.StorageInfo.MaxVolume
		productInfo.MaxIOPSVolume = json.StorageInfo.MaxIOPS
		productInfo.MaxThroughputVolume = "NA"
		productInfo.Description = "NA"

	case NETWORK_LOAD_BALANCER:
		json := (*jsonData).(NLBData)
		productInfo.ProductId = json.NLBName
		productInfo.RegionName = json.NLBInfo.RegionName
		productInfo.ZoneName = "NA"
		productInfo.Description = "NA"

	default:
		err := errors.New(productFamily + " is not supported product family!")
		cblogger.Error(err)
		return false, nil, err
	}

	hasKey := false
	checked := false
	if filterList != nil {
		// check filter
		i := interface{}(productInfo)
		hasKey, checked = checkFilters(&i, filterList)
		if hasKey {
			if !checked { // Has any key but not matched
				return hasKey, nil, nil
			}
		}
	}

	// filterList == nil or no policy Filter or checked == true
	// set CSPProductInfo after checking filter, because CSPProductInfo is not used for filtering
	switch productFamily {
	case COMPUTE_INSTANCE:
		json := (*jsonData).(InstanceData)
		productInfo.CSPProductInfo = json.InstanceInfo

	case STORAGE:
		json := (*jsonData).(StorageData)
		productInfo.CSPProductInfo = json.StorageInfo

	case NETWORK_LOAD_BALANCER:
		json := (*jsonData).(NLBData)
		productInfo.CSPProductInfo = json.NLBInfo

	default:
		err := errors.New(productFamily + " is not supported product family!")
		cblogger.Error(err)
		return hasKey, nil, err
	}

	return hasKey, &productInfo, nil
}

func checkFilters(jsonData *interface{}, filterList []irs.KeyValue) (hasKey bool, result bool) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called checkFilters()!")

	val := reflect.ValueOf(*jsonData)

	if val.Kind() != reflect.Struct {
		cblogger.Error("jsonData is not a struct")
		return false, false
	}

	// checking for all filter
	// if we have no filter, return true
	// if any field has a filter, it should be matched and then return true
	for _, filter := range filterList {
		matched := false
		for i := 0; i < val.NumField(); i++ {
			matched = false
			field := val.Type().Field(i)
			if strings.EqualFold(field.Name, filter.Key) {
				hasKey = true
				if str, ok := val.Field(i).Interface().(string); ok {
					if strings.EqualFold(str, filter.Value) {
						// filter matched
						matched = true
						cblogger.Debugln(field.Name+":", val.Field(i).Interface())
						break // of field for statement
					} else {
						continue // this field unmatched, so check next field
					}
				}
			}
		} // end of field for statement
		if !matched {
			return hasKey, false // filter unmatched, so exclude this product
		}
	} // end of filter for statement

	return hasKey, true
}

func transformToPriceInfo(productFamily string, jsonData *interface{}, filterList []irs.KeyValue) (bool /*hasKey*/, *irs.PriceInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called transformToPriceInfo()!")

	if jsonData == nil {
		err := errors.New("jsonData is nil")
		cblogger.Error(err)
		return false, nil, err
	}

	var priceList *PricingList

	switch productFamily {
	case COMPUTE_INSTANCE:
		json := (*jsonData).(InstanceData)
		priceList = &json.PricingList
	case STORAGE:
		json := (*jsonData).(StorageData)
		priceList = &json.PricingList
	case NETWORK_LOAD_BALANCER:
		json := (*jsonData).(NLBData)
		priceList = &json.PricingList
	default:
		err := errors.New(productFamily + " is not supported product family!")
		cblogger.Error(err)
		return false, nil, err
	}

	var priceInfo irs.PriceInfo

	// Transform PayAsYouGo를 to PricingPolicies
	if priceList.PayAsYouGo.PricingId != "" {
		paygPolicy := irs.PricingPolicies{
			PricingId:     priceList.PayAsYouGo.PricingId,
			PricingPolicy: "OnDemand", // "PayAsYouGo",
			Unit:          priceList.PayAsYouGo.Unit,
			Currency:      priceList.PayAsYouGo.Currency,
			Price:         priceList.PayAsYouGo.Price,
			Description:   "Pay-as-you-go pricing policy",
		}
		priceInfo.PricingPolicies = append(priceInfo.PricingPolicies, paygPolicy)
	}

	// Transform SavingPlan to PricingPolicies
	for _, plan := range priceList.SavingPlan {
		savingPolicy := irs.PricingPolicies{
			PricingId:     plan.PricingId,
			PricingPolicy: "Reserved", // "SavingPlan",
			Unit:          plan.Unit,
			Currency:      plan.Currency,
			Price:         plan.Price,
			Description:   plan.Term + " saving plan",
			PricingPolicyInfo: &irs.PricingPolicyInfo{
				LeaseContractLength: plan.Term,
				OfferingClass:       "NA",
				PurchaseOption:      "NA",
			},
		}
		priceInfo.PricingPolicies = append(priceInfo.PricingPolicies, savingPolicy)
	}

	if filterList == nil {
		priceInfo.CSPPriceInfo = priceList
		return false, &priceInfo, nil
	}

	gHasKey := false
	checked := false
	for _, policy := range priceInfo.PricingPolicies {
		// check filter
		i := interface{}(policy)
		hasKey := false
		hasKey, checked = checkFilters(&i, filterList)
		if hasKey {
			gHasKey = true
			if checked {
				break
			}
		} else {
			if policy.PricingPolicyInfo != nil {
				// and then checking for policy.PricingPolicyInfo struct
				// check filter
				i := interface{}(*policy.PricingPolicyInfo)
				hasKey := false
				hasKey, checked = checkFilters(&i, filterList)
				if hasKey {
					gHasKey = true
					if checked {
						break
					}
				}
			}
		}
	}

	if gHasKey {
		if !checked { // Has any key but not matched
			return gHasKey, nil, nil
		}
	}

	// filterList == nil or no policy Filter or checked == true
	// set CSPPriceInfo after checking filter, because CSPPriceInfo is not used for filtering
	priceInfo.CSPPriceInfo = priceList
	return gHasKey, &priceInfo, nil
}

func getMockPriceInfo(productFamily string, regionName string) ([]*interface{}, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called getMockPriceInfo()!")

	cbspiderRoot := os.Getenv("CBSPIDER_ROOT")
	if cbspiderRoot == "" {
		cblogger.Error("$CBSPIDER_ROOT is not set!!")
		os.Exit(1)
	}
	priceInfoDir := cbspiderRoot + "/cloud-control-manager/cloud-driver/drivers/mock/resources/price-info/mock-price-info"
	files, err := os.ReadDir(priceInfoDir)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	results := []*interface{}{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			filePath := priceInfoDir + "/" + file.Name()
			data, err := os.ReadFile(filePath)
			if err != nil {
				cblogger.Error(err)
				return nil, err
			}

			if productFamily == COMPUTE_INSTANCE && strings.Contains(file.Name(), "compute-instance") {
				var jsonData InstanceData
				err = json.Unmarshal(data, &jsonData)
				if err != nil {
					cblogger.Error(err)
					return nil, err
				}

				if regionName != "" && jsonData.InstanceInfo.RegionName == regionName {
					jsonDataInterface := interface{}(jsonData)
					results = append(results, &jsonDataInterface)
				}
			} else if productFamily == STORAGE && strings.Contains(file.Name(), "storage") {
				var jsonData StorageData
				err = json.Unmarshal(data, &jsonData)
				if err != nil {
					cblogger.Error(err)
					return nil, err
				}

				if regionName != "" && jsonData.StorageInfo.RegionName == regionName {
					jsonDataInterface := interface{}(jsonData)
					results = append(results, &jsonDataInterface)
				}
			} else if productFamily == NETWORK_LOAD_BALANCER && strings.Contains(file.Name(), "network-loadbalancer") {
				var jsonData NLBData
				err = json.Unmarshal(data, &jsonData)
				if err != nil {
					cblogger.Error(err)
					return nil, err
				}

				if regionName != "" && jsonData.NLBInfo.RegionName == regionName {
					jsonDataInterface := interface{}(jsonData)
					results = append(results, &jsonDataInterface)
				}
			}
		}
	}

	return results, nil
}

func GetGlobalViewTemplate(productFamily string) (irs.CloudPriceData, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetGlobalViewTemplate()!")

	cbspiderRoot := os.Getenv("CBSPIDER_ROOT")
	if cbspiderRoot == "" {
		cblogger.Error("$CBSPIDER_ROOT is not set!!")
		os.Exit(1)
	}
	priceInfoDir := cbspiderRoot + "/cloud-control-manager/cloud-driver/drivers/mock/resources/price-info/global-view-template"
	files, err := os.ReadDir(priceInfoDir)
	if err != nil {
		cblogger.Error(err)
		return irs.CloudPriceData{}, err
	}

	productFileName := ""
	switch productFamily {
	case COMPUTE_INSTANCE:
		productFileName = "compute-instance"
	case STORAGE:
		productFileName = "storage"
	case NETWORK_LOAD_BALANCER:
		productFileName = "network-loadbalancer"
	default:
		err := errors.New(productFamily + " is not supported product family!")
		cblogger.Error(err)
		return irs.CloudPriceData{}, err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") && strings.Contains(file.Name(), productFileName) {
			filePath := priceInfoDir + "/" + file.Name()
			data, err := os.ReadFile(filePath)
			if err != nil {
				cblogger.Error(err)
				return irs.CloudPriceData{}, err
			}

			var cloudPriceData irs.CloudPriceData
			err = json.Unmarshal(data, &cloudPriceData)
			if err != nil {
				cblogger.Error(err)
				return irs.CloudPriceData{}, err
			}

			return cloudPriceData, nil
		}
	}

	err = errors.New(productFamily + " has not Global View Template!")
	cblogger.Error(err)
	return irs.CloudPriceData{}, err
}
