package resources

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentPriceInfoHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

type TencentInstanceModel struct {
	standardInfo *cvm.DescribeZoneInstanceConfigInfosResponse
	reservedInfo *cvm.DescribeReservedInstancesConfigInfosResponse
}

type TencentInstanceInformation struct {
	PriceList      *irs.Price
	StandardPrices *[]TencentCommonInstancePrice
	ReservedPrices *[]TencentReservedInstancePrice
}

type TencentCommonInstancePrice struct {
	InstanceChargeType *string
	Price              *cvm.ItemPrice
}

type TencentReservedInstancePrice struct {
	Price *cvm.ReservedInstancePriceItem
}

func (t *TencentPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	pl := []string{"cvm"}
	return pl, nil
}

func (t *TencentPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, additionalFilters []irs.KeyValue) (string, error) {

	filterKeyValueMap := mapToTencentFilter(additionalFilters)

	switch {
	case strings.EqualFold("cvm", productFamily):
		if t.Client.GetRegion() != regionName {
			t.Client.Init(regionName)
		}
		keyValueMap := make(map[string]string)
		for _, kv := range additionalFilters {
			keyValueMap[kv.Key] = kv.Value
		}

		standardInfo, err := describeZoneInstanceConfigInfos(t.Client, filterKeyValueMap)
		if err != nil {
			return "", err
		}

		res, err := mappingToComputeStruct(t.Client.GetRegion(), standardInfo, keyValueMap)
		if err != nil {
			return "", err
		}

		parsedResponse, err := convertJsonStringNoEscape(res)
		if err != nil {
			return "", err
		}

		return parsedResponse, nil
	}

	return "", nil
}

func convertJsonStringNoEscape(v interface{}) (string, error) {
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

func describeZoneInstanceConfigInfos(client *cvm.Client, filterMap map[string]*cvm.Filter) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
	filters := parseToFilterSlice(filterMap, "zoneName", "instanceFamily", "instanceType")

	optionFilters := parseToFilterSlice(filterMap, "instance-charge-type")
	if len(optionFilters) > 0 {
		filters = append(filters, optionFilters...)
	} else {
		hourlyFilter := &cvm.Filter{
			Name:   common.StringPtr("instance-charge-type"),
			Values: []*string{common.StringPtr("POSTPAID_BY_HOUR")},
		}
		filters = append(filters, hourlyFilter)
	}

	req := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	req.Filters = filters

	res, err := client.DescribeZoneInstanceConfigInfos(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func mappingToComputeStruct(regionName string, standardInfo *cvm.DescribeZoneInstanceConfigInfosResponse, filterMap map[string]string) (*irs.CloudPrice, error) {
	priceMap := make(map[string]irs.Price)

	if standardInfo != nil {
		for _, v := range standardInfo.Response.InstanceTypeQuotaSet {
			if *v.InstanceChargeType != "POSTPAID_BY_HOUR" {
				continue
			}

			productId := computeInstanceKeyGeneration(*v.Zone, *v.InstanceType, *v.CpuType, strconv.FormatInt(*v.Memory, 10))

			price, ok := priceMap[productId]
			if ok {
				onDemand := mappingOnDemand(v.InstanceChargeType, v.Price)

				if priceFilter(&onDemand, filterMap) {
					continue
				}

				price.PriceInfo.OnDemand = onDemand
				price.PriceInfo.CSPPriceInfo = *v.Price
				priceMap[productId] = price

			} else {
				productInfo := mappingProductInfo(regionName, *v)
				if productFilter(filterMap, &productInfo) {
					continue
				}

				onDemand := mappingOnDemand(v.InstanceChargeType, *v.Price)

				if priceFilter(&onDemand, filterMap) {
					continue
				}

				aPrice := irs.Price{}
				aPrice.ZoneName = *v.Zone
				priceInfo := irs.PriceInfo{}
				priceInfo.CSPPriceInfo = *v.Price
				priceInfo.OnDemand = onDemand

				aPrice.ProductInfo = productInfo
				aPrice.PriceInfo = priceInfo

				priceMap[productId] = aPrice
			}
		}
	}

	priceList := make([]irs.Price, 0)
	if priceMap != nil && len(priceMap) > 0 {
		for _, priceValue := range priceMap {
			priceList = append(priceList, priceValue)
		}
	}

	cloudPrice := &irs.CloudPrice{
		Meta:       irs.Meta{Version: "0.5", Description: "Multi-Cloud Price Info"},
		CloudName:  "TENCENT",
		RegionName: regionName,
		PriceList:  priceList,
	}

	return cloudPrice, nil
}

func productFilter(filterMap map[string]string, productInfo *irs.ProductInfo) bool {
	if len(filterMap) <= 0 {
		return false
	}

	if value, ok := filterMap["instanceType"]; ok && value != "" && value != (*productInfo).VMSpecInfo.Name {
		return true
	}

	if value, ok := filterMap["vcpu"]; ok && value != "" && value != (*productInfo).VMSpecInfo.VCpu.Count {
		return true
	}

	if value, ok := filterMap["memory"]; ok && value != "" && value != (*productInfo).VMSpecInfo.MemSizeMiB {
		return true
	}

	if value, ok := filterMap["gpu"]; ok && value != "" {
		if len((*productInfo).VMSpecInfo.Gpu) <= 0 {
			return true
		}
		if value != (*productInfo).VMSpecInfo.Gpu[0].Count {
			return true
		}
	}
	if value, ok := filterMap["storage"]; ok && value != "" && value != (*productInfo).VMSpecInfo.DiskSizeGB {
		return true
	}
	return false
}

func priceFilter(policy *irs.OnDemand, filterMap map[string]string) bool {
	if len(filterMap) <= 0 {
		return false
	}
	if value, ok := filterMap["pricingId"]; ok && value != (*policy).PricingId {
		return true
	}
	if value, ok := filterMap["unit"]; ok && value != "" && value != (*policy).Unit {
		return true
	}
	if value, ok := filterMap["price"]; ok && value != "" && value != (*policy).Price {
		return true
	}
	if value, ok := filterMap["currency"]; ok && value != "" && value != (*policy).Currency {
		return true
	}
	if value, ok := filterMap["description"]; ok && value != "" && value != (*policy).Description {
		return true
	}
	return false
}

func mappingProductInfo(regionName string, i interface{}) irs.ProductInfo {
	productInfo := irs.ProductInfo{
		CSPProductInfo: i,
	}

	switch v := i.(type) {
	case cvm.InstanceTypeQuotaItem:
		vm := i.(cvm.InstanceTypeQuotaItem)
		productInfo.ProductId = *vm.Zone + "_" + *vm.InstanceType

		simpleMode := strings.ToUpper(os.Getenv("VMSPECINFO_SIMPLE_MODE_IN_PRICEINFO")) == "ON"

		if simpleMode {
			productInfo.VMSpecName = strPtrNilCheck(vm.InstanceType)
		} else {
			vmSpecInfo := irs.VMSpecInfo{
				Name:       strPtrNilCheck(vm.InstanceType),
				VCpu:       irs.VCpuInfo{Count: intPtrNilCheck(vm.Cpu), ClockGHz: extractClockValue(*vm.Frequency)},
				MemSizeMiB: irs.ConvertGiBToMiBInt64(*vm.Memory),
				DiskSizeGB: "-1",
			}

			if int(*vm.Gpu) > 0 {
				vmSpecInfo.Gpu = []irs.GpuInfo{
					{
						Count:          strconv.Itoa(int(*vm.Gpu)),
						MemSizeGB:      "-1",
						TotalMemSizeGB: "-1",
						Mfr:            "NA",
						Model:          "NA",
					},
				}
			}

			productInfo.VMSpecInfo = &vmSpecInfo
		}

		productInfo.Description = strPtrNilCheck(vm.CpuType) + ", " + strPtrNilCheck(vm.Remark)

		return productInfo

	default:
		cblogger.Debug(v)
	}

	return irs.ProductInfo{}
}

func extractClockValue(frequencyStr string) string {
	if frequencyStr == "" {
		return ""
	}

	var clockValue string

	if idx := strings.Index(frequencyStr, "/"); idx >= 0 {
		firstPart := strings.TrimSpace(frequencyStr[:idx])
		secondPart := strings.TrimSpace(frequencyStr[idx+1:])

		if firstPart == "" || firstPart == "-" {
			clockValue = secondPart
		} else {
			clockValue = firstPart
		}
	} else {
		clockValue = frequencyStr
	}

	clockValue = strings.TrimSuffix(clockValue, "GHz")

	return clockValue
}

func mappingOnDemand(instanceChargeType *string, price any) irs.OnDemand {
	policy := irs.OnDemand{
		PricingId:   "NA",
		Currency:    "USD",
		Description: "NA",
	}

	objType := reflect.TypeOf(price)
	isPointer := false

	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
		isPointer = true
	}
	switch objType {
	case reflect.TypeOf(cvm.ItemPrice{}):
		if isPointer {
			price = reflect.ValueOf(price).Elem().Interface()
		}
		p := price.(cvm.ItemPrice)

		policy.Unit = "Hour" // strPtrNilCheck(p.ChargeUnit)
		policy.Price = floatPtrNilCheck(p.UnitPriceDiscount)

	default:
		cblogger.Info("Type doesn't match", reflect.TypeOf(price))
	}

	return policy
}

func computeInstanceKeyGeneration(hashingKeys ...string) string {
	keys := ""
	for _, key := range hashingKeys {
		if len(strings.TrimSpace(key)) > 0 {
			keys += strings.TrimSpace(key)
		}
	}
	return keys
}

func mapToTencentFilter(additionalFilterList []irs.KeyValue) map[string]*cvm.Filter {
	filterMap := make(map[string]*cvm.Filter, 0)

	for _, kv := range additionalFilterList {
		switch kv.Key {
		case "zoneName":
			filterMap[kv.Key] = &cvm.Filter{
				Name:   common.StringPtr("zone"),
				Values: []*string{common.StringPtr(kv.Value)},
			}
		case "instanceType":
			filterMap[kv.Key] = &cvm.Filter{
				Name:   common.StringPtr("instance-type"),
				Values: []*string{common.StringPtr(kv.Value)},
			}
		case "instanceFamily":
			filterMap[kv.Key] = &cvm.Filter{
				Name:   common.StringPtr("instance-family"),
				Values: []*string{common.StringPtr(kv.Value)},
			}
		default:
			filterMap[kv.Key] = &cvm.Filter{
				Name:   common.StringPtr(kv.Key),
				Values: []*string{common.StringPtr(kv.Value)},
			}
		}
	}
	return filterMap
}

func parseToFilterSlice(filterMap map[string]*cvm.Filter, conditions ...string) []*cvm.Filter {
	var filters []*cvm.Filter
	for _, condition := range conditions {
		if val, ok := filterMap[condition]; ok {
			filters = append(filters, val)
		}
	}

	return filters
}
