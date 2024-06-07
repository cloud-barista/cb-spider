package resources

import (
	"bytes"
	"encoding/json"
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
		parsedResponse, err := convertJsonStringNoEscape(&res)
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

func mappingToComputeStruct(regionName string, instanceModel *TencentInstanceModel, filterMap map[string]string) (*irs.CloudPriceData, error) {
	priceMap := make(map[string]irs.Price) // productinfo , priceinfo

	// standardInfo
	if instanceModel.standardInfo != nil {
		for _, v := range instanceModel.standardInfo.Response.InstanceTypeQuotaSet {

			productId := computeInstanceKeyGeneration(*v.Zone, *v.InstanceType, *v.CpuType, strconv.FormatInt(*v.Memory, 10))
			price, ok := priceMap[productId]
			if ok { // 있으면
				// policies 추출
				policy := mappingPricingPolicy(v.InstanceChargeType, v.Price)

				if priceFilter(&policy, filterMap) {
					continue
				}
				// append policy
				pricePolicies := price.PriceInfo.PricingPolicies
				pricePolicies = append(pricePolicies, policy)
				price.PriceInfo.PricingPolicies = pricePolicies
				priceMap[productId] = price // price 재할당

			} else { // 없으면
				// product 추출
				productInfo := mappingProductInfo(regionName, *v)
				if productFilter(filterMap, &productInfo) {
					continue
				}

				// pricePolicies 추출
				policy := mappingPricingPolicy(v.InstanceChargeType, *v.Price)

				if priceFilter(&policy, filterMap) {
					continue
				}

				aPrice := irs.Price{}
				priceInfo := irs.PriceInfo{}

				pricePolicies := []irs.PricingPolicies{}
				pricePolicies = append(pricePolicies, policy)

				priceInfo.PricingPolicies = pricePolicies

				aPrice.ProductInfo = productInfo
				aPrice.PriceInfo = priceInfo

				priceMap[productId] = aPrice
			}
		} // end of for
	}
	// reservedInfo
	if instanceModel.reservedInfo != nil {
		for _, v := range instanceModel.reservedInfo.Response.ReservedInstanceConfigInfos {
			for _, info := range v.InstanceFamilies {
				for _, iType := range info.InstanceTypes {
					for _, p := range iType.Prices {
						productId := computeInstanceKeyGeneration(*p.Zone, *iType.InstanceType, *iType.CpuModelName, strconv.FormatUint(*iType.Memory, 10))

						price, ok := priceMap[productId]
						if ok { // 있으면
							// policies 추출
							policy := mappingPricingPolicy(common.StringPtr("RESERVED"), p)

							if priceFilter(&policy, filterMap) {
								continue
							}

							// append policy
							pricePolicies := price.PriceInfo.PricingPolicies
							pricePolicies = append(pricePolicies, policy)
							price.PriceInfo.PricingPolicies = pricePolicies
							priceMap[productId] = price // price 재할당
						} else { // 없으면
							// product 추출
							productInfo := mappingProductInfo(regionName, *iType)
							if productFilter(filterMap, &productInfo) {
								continue
							}

							//	for _, val := range *v.price.Res {

							// pricePolicies 추출
							policy := mappingPricingPolicy(common.StringPtr("RESERVED"), *p)

							if priceFilter(&policy, filterMap) {
								continue
							}
							aPrice := irs.Price{}
							priceInfo := irs.PriceInfo{}

							pricePolicies := []irs.PricingPolicies{}
							pricePolicies = append(pricePolicies, policy)

							priceInfo.PricingPolicies = pricePolicies

							aPrice.ProductInfo = productInfo
							aPrice.PriceInfo = priceInfo

							priceMap[productId] = aPrice
							//	}
						}

					} // end of itype.Prices for
				} // end of itype for
			} // end of instanceFamilies for
		} // end of reservedInstanceConfigInfos for
	}

	priceList := make([]irs.Price, 0)
	if priceMap != nil && len(priceMap) > 0 {
		for _, priceValue := range priceMap {
			priceList = append(priceList, priceValue)
		}
	}

	cloudPriceData := &irs.CloudPriceData{
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
	return cloudPriceData, nil

}

// product 항목에 대해 필터 맵에 값이 있으면 true반환 -> true면 해당 값 필터링
func productFilter(filterMap map[string]string, productInfo *irs.ProductInfo) bool {
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

// price 항목에 대해 필터 맵에 값이 있으면 true반환 -> true면 해당 값 필터링
func priceFilter(policy *irs.PricingPolicies, filterMap map[string]string) bool {
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
	if value, ok := filterMap["purchaseOption"]; ok && value != "" && value != (*policy.PricingPolicyInfo).PurchaseOption {
		return true
	}
	if value, ok := filterMap["purchaseOption"]; ok && value != "" && value != (*policy.PricingPolicyInfo).PurchaseOption {
		return true
	}
	if value, ok := filterMap["leaseContractLength"]; ok && value != "" && value != (*policy.PricingPolicyInfo).LeaseContractLength {
		return true
	}
	return false
}

// TencentSDK VM Product & Pricing struct to irs ProductPolicies
// storage 출력 항목 삭제
// compute infra 관련 정보만 매핑
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

		// storage 관련 정보 삭제
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

		// storage 관련 정보 삭제
		return productInfo
	default:
		cblogger.Debug(v)
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
	// POSTPAID -> v20170312.ItemPrice 반환 / SPOTPAID -> *v20170312.ItemPrice 포인터 반환
	// 포인터가 가리키는 실제 타입을 확인하여 포인터와 비 포인터를 동일하게 처리하기 위함
	objType := reflect.TypeOf(price)
	isPointer := false

	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
		isPointer = true
	}
	switch objType {
	case reflect.TypeOf(cvm.ItemPrice{}):
		// 포인터일 경우, 실제 값을 가져온다
		if isPointer {
			price = reflect.ValueOf(price).Elem().Interface()
		}
		p := price.(cvm.ItemPrice)

		policy.Unit = strPtrNilCheck(p.ChargeUnit)
		policy.Price = floatPtrNilCheck(p.UnitPrice)

		// NA
		policy.Description = strPtrNilCheck(nil)

		policyInfo.LeaseContractLength = strPtrNilCheck(nil)
		policyInfo.OfferingClass = strPtrNilCheck(nil)
		policyInfo.PurchaseOption = strPtrNilCheck(nil)

	case reflect.TypeOf(cvm.ReservedInstancePriceItem{}):
		if isPointer {
			price = reflect.ValueOf(price).Elem().Interface()
		}
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
		//cblogger.Debug(objType)
		cblogger.Info("Type doesn't match", reflect.TypeOf(price))
	}

	return policy
}

// Instance Type 별 고유 key 생성
func computeInstanceKeyGeneration(hashingKeys ...string) string {
	// h := fnv.New32a()

	keys := ""
	for _, key := range hashingKeys {
		if len(strings.TrimSpace(key)) > 0 {
			keys += strings.TrimSpace(key)
			// _, err := h.Write([]byte(key))
			// if err != nil {
			// 	return ""
			// }
		}
	}
	return keys

	//return strconv.FormatUint(uint64(h.Sum32()), 10)
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
