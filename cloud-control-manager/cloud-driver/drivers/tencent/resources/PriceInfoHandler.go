package resources

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"strconv"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
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

// ListProductFamily
// tencent 에서 제공해주는 product family 관련한 api 가 존재하지 않아 코드 레벨에서 관리하도록 1차 논의 완료
// 2023.12.14.driver 내부에서 array 등으로 관리하는 방침으로 변경
func (t *TencentPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	//pl := []string{"cvm", "k8s", "cbm", "gpu"}
	pl := []string{"cvm"}
	//pl := make([]string, 0)
	return pl, nil
}

func (t *TencentPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, additionalFilters []irs.KeyValue) (string, error) {

	filterKeyValueMap := mapToTencentFilter(additionalFilters)

	switch {
	case strings.EqualFold("cvm", productFamily):
		// client 생성 with region name
		if t.Client.GetRegion() != regionName {
			t.Client.Init(regionName)
		}
		keyValueMap := make(map[string]string)
		for _, kv := range additionalFilters {
			keyValueMap[kv.Key] = kv.Value
		}

		//Common Instance Price calculator
		standardInfo, err := describeZoneInstanceConfigInfos(t.Client, filterKeyValueMap)

		if err != nil {
			return "", err
		}

		//Reserved Instance Price calculator
		reservedInfo, err := describeReservedInstancesConfigInfos(t.Client, filterKeyValueMap)

		if err != nil {
			return "", err
		}

		res, err := mappingToComputeStruct(t.Client.GetRegion(), &TencentInstanceModel{standardInfo: standardInfo, reservedInfo: reservedInfo}, keyValueMap)

		if err != nil {
			return "", err
		}
		//cblogger.Info(*res)
		parsedResponse, err := convertJsonStringNoEscape(&res)
		// mar, err := json.Marshal(&res)
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

/*
AZ 의 Instance standard 모델과 Spot 모델 조회
*/
func describeZoneInstanceConfigInfos(client *cvm.Client, filterMap map[string]*cvm.Filter) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
	//Filter 정적 추가
	filters := parseToFilterSlice(filterMap, "zoneName", "instanceFamily", "instanceType") //필수

	optionFilters := parseToFilterSlice(filterMap, "instance-charge-type") // option
	if len(optionFilters) > 0 {
		filters = append(filters, optionFilters...)
	}

	//신규 Instance 데이터 Request 생성
	req := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	req.Filters = filters

	//인스턴스 정보 조회
	res, err := client.DescribeZoneInstanceConfigInfos(req)
	if err != nil {
		// TODO Error mapping
		return nil, err
	}

	return res, nil
}

// TODO 기존, 신규 인스턴스 가격과 예약 인스턴스 가격 조회 후, 통합기능 개발
// Issue
// 1. 신규 인스턴스와 예약 인스턴스 간의 require param & recommand param 이 달라, 별개의 필터 구현이 필요
// 2. return value가 서로 달라, 1개의 통일된 struct array로 변환과정이 복잡함

// Reserved Instance Price calculator
func describeReservedInstancesConfigInfos(client *cvm.Client, filterMap map[string]*cvm.Filter) (*cvm.DescribeReservedInstancesConfigInfosResponse, error) {
	filters := parseToFilterSlice(filterMap, "zoneName")
	optionFilters := parseToFilterSlice(filterMap, "instance-charge-type") // option
	req := cvm.NewDescribeReservedInstancesConfigInfosRequest()
	req.Filters = filters

	if len(optionFilters) > 0 {
		filters = append(filters, optionFilters...)
	}
	res, err := client.DescribeReservedInstancesConfigInfos(req)
	if err != nil {
		// TODO Error mapping
		return nil, err
	}

	return res, nil
}

// Mapper Start Function
func mappingToComputeStruct(regionName string, instanceModel *TencentInstanceModel, filterMap map[string]string) (*irs.CloudPriceData, error) {
	priceMap := make(map[string]*TencentInstanceInformation, 0)

	if instanceModel.standardInfo != nil {
		for _, v := range instanceModel.standardInfo.Response.InstanceTypeQuotaSet {

			//변수값이 충분한지 고려할 필요가 있음, reservedInstance ReturnValue와 비교하여, 최대한 고유하게 가져갈 수 있는 것들은
			//가져가도록 수정
			key := computeInstanceKeyGeneration(*v.Zone, *v.InstanceType, *v.CpuType, strconv.FormatInt(*v.Memory, 10))

			if pp, ok := priceMap[key]; !ok {
				productInfo := mappingProductInfo(regionName, *v)

				if validateFilter(filterMap, &productInfo) {
					continue
				}
				sp := make([]TencentCommonInstancePrice, 0)
				sp = append(sp, TencentCommonInstancePrice{InstanceChargeType: v.InstanceChargeType, Price: v.Price})

				priceMap[key] = &TencentInstanceInformation{
					PriceList: &irs.Price{
						ProductInfo: productInfo,
					},
					StandardPrices: &sp,
				}
			} else {
				newSlice := append(*pp.StandardPrices, TencentCommonInstancePrice{InstanceChargeType: v.InstanceChargeType, Price: v.Price})
				pp.StandardPrices = &newSlice
			}
		}
	}

	//TODO reserved Instance Info Mapping
	if instanceModel.reservedInfo != nil {
		for _, v := range instanceModel.reservedInfo.Response.ReservedInstanceConfigInfos {
			for _, info := range v.InstanceFamilies {
				for _, iType := range info.InstanceTypes {
					for _, p := range iType.Prices {
						key := computeInstanceKeyGeneration(*p.Zone, *iType.InstanceType, *iType.CpuModelName, strconv.FormatUint(*iType.Memory, 10))
						if pp, ok := priceMap[key]; !ok {
							productInfo := mappingProductInfo(regionName, *iType)

							if validateFilter(filterMap, &productInfo) {
								continue
							}

							rp := make([]TencentReservedInstancePrice, 0)
							rp = append(rp, TencentReservedInstancePrice{Price: p})

							priceMap[key] = &TencentInstanceInformation{
								PriceList: &irs.Price{
									ProductInfo: productInfo,
								},

								ReservedPrices: &rp,
							}
						} else {
							newSlice := append(*pp.ReservedPrices, TencentReservedInstancePrice{Price: p})
							pp.ReservedPrices = &newSlice
						}
					}

				}
			}
		}
	}
	generatePriceInfo(priceMap, filterMap)

	priceList := make([]irs.Price, 0)

	if priceMap != nil && len(priceMap) > 0 {
		for _, v := range priceMap {
			priceList = append(priceList, *v.PriceList)
		}
	}

	x := &irs.CloudPriceData{
		Meta: irs.Meta{
			Version:     "v0.1",
			Description: "Multi-Cloud Price Info Api",
		},
		CloudPriceList: []irs.CloudPrice{
			{
				CloudName: "TENCENT",
				PriceList: priceList,
			},
		},
	}
	return x, nil
}

// TencentSDK VM Product & Pricing struct to irs Struct
func generatePriceInfo(priceMap map[string]*TencentInstanceInformation, filterMap map[string]string) {
	if priceMap != nil && len(priceMap) > 0 {
		for _, v := range priceMap {
			pl := v.PriceList
			policies := make([]irs.PricingPolicies, 0)
			prices := make([]any, 0)

			if v.StandardPrices != nil && len(*v.StandardPrices) > 0 {
				for _, val := range *v.StandardPrices {

					policy := mappingPricingPolicy(val.InstanceChargeType, *val.Price)

					if priceValidateFilter(&policy, filterMap) {
						continue
					}
					prices = append(prices, val.Price)
					policies = append(policies, policy)
				}
			}

			if v.ReservedPrices != nil && len(*v.ReservedPrices) > 0 {
				for _, val := range *v.ReservedPrices {
					policy := mappingPricingPolicy(common.StringPtr("RESERVED"), *val.Price)

					if priceValidateFilter(&policy, filterMap) {
						continue
					}
					prices = append(prices, val.Price)
					policies = append(policies, policy)
				}
			}
			// //mar, err := json.Marshal(prices)
			// mar, err := ConvertJsonStringNoEscape(prices)

			// if err != nil {
			// 	continue
			// }

			pl.PriceInfo = irs.PriceInfo{
				PricingPolicies: policies,
				CSPPriceInfo:    prices,
			}
		}
	}
}
func validateFilter(filterMap map[string]string, productInfo *irs.ProductInfo) bool {
	if len(filterMap) <= 0 {
		return false
	}

	if value, ok := filterMap["zoneName"]; ok && value != "" && value != (*productInfo).ZoneName {
		return true
	}

	if value, ok := filterMap["instanceType"]; ok && value != "" && value != (*productInfo).InstanceType {
		return true
	}

	if value, ok := filterMap["vcpu"]; ok && value != "" && value != (*productInfo).Vcpu {
		return true
	}

	if value, ok := filterMap["memory"]; ok && value != "" && value != (*productInfo).Memory {
		return true
	}

	if value, ok := filterMap["gpu"]; ok && value != "" && value != (*productInfo).Gpu {
		return true
	}
	if value, ok := filterMap["storage"]; ok && value != "" && value != (*productInfo).Storage {
		return true
	}
	return false
}

func priceValidateFilter(policy *irs.PricingPolicies, filterMap map[string]string) bool {
	if len(filterMap) <= 0 {
		return false
	}
	if value, ok := filterMap["pricingId"]; ok && value != (*policy).PricingId {
		return true
	}
	// filter[unit] = ChargeUnit key값 존재확인 HOUR 값 넣어줌 빈값아님 같은값일경우 false 같은값일때 ㅇ
	if value, ok := filterMap["unit"]; ok && value != "" && value != (*policy).Unit {
		return true
	}
	// filter[price] = UnitPrice
	if value, ok := filterMap["price"]; ok && value != "" && value != (*policy).Price {
		return true
	}
	if value, ok := filterMap["currency"]; ok && value != "" && value != (*policy).Currency {
		return true
	}
	if value, ok := filterMap["description"]; ok && value != "" && value != (*policy).Description {
		return true
	}
	if value, ok := filterMap["unit"]; ok && value != "" && value != (*policy).Unit {
		return true
	}
	if value, ok := filterMap["purchaseOption"]; ok && value != "" && value != (*policy.PricingPolicyInfo).PurchaseOption {
		return true
	}
	if value, ok := filterMap["purchaseOption"]; ok && value != "" && value != (*policy.PricingPolicyInfo).PurchaseOption {
		return true
	}
	return false
}

func resPriceValidateFilter(policy *irs.PricingPolicies, filterMap map[string]string) bool {
	if len(filterMap) <= 0 {
		return false
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
	if value, ok := filterMap["unit"]; ok && value != "" && value != (*policy).Unit {
		return true
	}
	if value, ok := filterMap["purchaseOption"]; ok && value != "" && value != (*policy.PricingPolicyInfo).PurchaseOption {
		return true
	}
	if value, ok := filterMap["purchaseOption"]; ok && value != "" && value != (*policy.PricingPolicyInfo).PurchaseOption {
		return true
	}

	return false
}

// 	if len(filter) <= 0 {
// 		return false
// 	}
// 	if value, ok := filter["price"]; ok && value != "" && value != police.Price {
// 		return true
// 	}
// 	if value, ok := filter["currency"]; ok && value != "" && value != police.Currency {
// 		return true
// 	}
// 	if value, ok := filter["unit"]; ok && value != "" && value != police.Unit {
// 		return true
// 	}
// 	return false
// }

// TencentSDK VM Product & Pricing struct to irs ProductPolicies
func mappingProductInfo(regionName string, i interface{}) irs.ProductInfo {
	productInfo := irs.ProductInfo{
		//ProductId:      "NA",
		RegionName:     regionName,
		CSPProductInfo: i,
	}

	switch v := i.(type) {
	case cvm.InstanceTypeQuotaItem:
		vm := i.(cvm.InstanceTypeQuotaItem)
		productInfo.ProductId = regionName + "-" + *vm.InstanceType
		productInfo.InstanceType = strPtrNilCheck(vm.InstanceType)
		productInfo.ZoneName = *vm.Zone
		productInfo.Vcpu = intPtrNilCheck(vm.Cpu)
		productInfo.Memory = intPtrNilCheck(vm.Memory)
		productInfo.Gpu = intPtrNilCheck(vm.Gpu)
		productInfo.Description = strPtrNilCheck(vm.CpuType)

		// not provide from tencent
		productInfo.Storage = intPtrNilCheck(vm.StorageBlockAmount)
		productInfo.GpuMemory = strPtrNilCheck(nil)
		productInfo.OperatingSystem = strPtrNilCheck(nil)
		productInfo.PreInstalledSw = strPtrNilCheck(nil)

		// not suit for compute instance
		productInfo.VolumeType = strPtrNilCheck(nil)
		productInfo.StorageMedia = strPtrNilCheck(nil)
		productInfo.MaxVolumeSize = strPtrNilCheck(nil)
		productInfo.MaxIOPSVolume = strPtrNilCheck(nil)
		productInfo.MaxThroughputVolume = strPtrNilCheck(nil)

		return productInfo

	case cvm.ReservedInstanceTypeItem:
		reservedVm := i.(cvm.ReservedInstanceTypeItem)
		productInfo.ProductId = regionName + "-" + *reservedVm.InstanceType
		productInfo.InstanceType = strPtrNilCheck(reservedVm.InstanceType)
		productInfo.ZoneName = *reservedVm.Prices[0].Zone
		productInfo.Vcpu = uintPtrNilCheck(reservedVm.Cpu)
		productInfo.Memory = uintPtrNilCheck(reservedVm.Memory)
		productInfo.Gpu = uintPtrNilCheck(reservedVm.Gpu)
		productInfo.Description = strPtrNilCheck(reservedVm.CpuModelName)

		// not provide from tencent
		productInfo.Storage = strPtrNilCheck(nil)
		productInfo.GpuMemory = strPtrNilCheck(nil)
		productInfo.OperatingSystem = strPtrNilCheck(nil)
		productInfo.PreInstalledSw = strPtrNilCheck(nil)

		// not suit for compute instance
		productInfo.VolumeType = strPtrNilCheck(nil)
		productInfo.StorageMedia = strPtrNilCheck(nil)
		productInfo.MaxVolumeSize = strPtrNilCheck(nil)
		productInfo.MaxIOPSVolume = strPtrNilCheck(nil)
		productInfo.MaxThroughputVolume = strPtrNilCheck(nil)

		return productInfo
	default:
		spew.Dump(v)
	}

	return irs.ProductInfo{}

}

// TencentSDK VM Product & Pricing struct to irs PricingPolicies
func mappingPricingPolicy(instanceChargeType *string, price any) irs.PricingPolicies {

	// price info mapping
	policyInfo := irs.PricingPolicyInfo{}

	policy := irs.PricingPolicies{
		PricingId:         "NA",
		PricingPolicy:     *instanceChargeType,
		Currency:          "USD",
		PricingPolicyInfo: &policyInfo,
	}

	switch v := price.(type) {
	case cvm.ItemPrice:
		p := price.(cvm.ItemPrice)
		policy.Unit = strPtrNilCheck(p.ChargeUnit)
		policy.Price = floatPtrNilCheck(p.UnitPrice)

		// NA
		policy.Description = strPtrNilCheck(nil)

		policyInfo.LeaseContractLength = strPtrNilCheck(nil)
		policyInfo.OfferingClass = strPtrNilCheck(nil)
		policyInfo.PurchaseOption = strPtrNilCheck(nil)

	case cvm.ReservedInstancePriceItem:
		p := price.(cvm.ReservedInstancePriceItem)
		policy.PricingId = strPtrNilCheck(p.ReservedInstancesOfferingId)
		policy.Unit = strPtrNilCheck(common.StringPtr("Yrs"))
		policy.Price = floatPtrNilCheck(p.FixedPrice)
		policy.Description = strPtrNilCheck(p.ProductDescription)

		// 31536000 -> 1년
		var duration *uint64
		if p.Duration != nil {
			duration = p.Duration
		} else {
			duration = common.Uint64Ptr(0)
		}
		policyInfo.LeaseContractLength = strconv.FormatUint(*duration/31536000, 32) + "Yrs" // duration 초로 넘어옴 이거를 연도로 환산
		policyInfo.PurchaseOption = strPtrNilCheck(p.OfferingType)

		// NA
		policyInfo.OfferingClass = strPtrNilCheck(nil)

	default:
		spew.Dump(v)
	}

	return policy
}

// Instance Type 별 고유 key 생성
func computeInstanceKeyGeneration(hashingKeys ...string) string {
	h := fnv.New32a()

	for _, key := range hashingKeys {
		if len(strings.TrimSpace(key)) > 0 {
			_, err := h.Write([]byte(key))
			if err != nil {
				return ""
			}
		}
	}
	return strconv.FormatUint(uint64(h.Sum32()), 10)
}

// function 에 대한 explain 추가 작성
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
