package resources

import (
	"encoding/json"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentPriceInfoHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

type instanceModel struct {
	standardInfo *cvm.DescribeZoneInstanceConfigInfosResponse
	reservedInfo *cvm.DescribeReservedInstancesConfigInfosResponse
}

type productAndPrice struct {
	PriceList      *irs.PriceList
	StandardPrices *[]standardVmPrice
	ReservedPrices *[]reservedVmPrice
}

type standardVmPrice struct {
	InstanceChargeType *string
	Price              *cvm.ItemPrice
}

type reservedVmPrice struct {
	Price *cvm.ReservedInstancePriceItem
}

// ListProductFamily
// tencent 에서 제공해주는 product family 관련한 api 가 존재하지 않아 코드 레벨에서 관리하도록 1차 논의 완료 -> 이 매니저님: 23/12/13
func (t *TencentPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	pl := make([]string, 0)
	return pl, nil
}

func (t *TencentPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, additionalFilters []irs.KeyValue) (string, error) {

	filterMap := mapToFilter(additionalFilters)

	switch {
	case strings.EqualFold("compute", productFamily):

		// client 생성 with region name
		client, err := createClientByRegionName(t.Client.GetCredential(), regionName, t.Region.Region)
		// TODO 신 매니저님 : connection 의 region 논의 후 코드 수정
		//client := t.Client.Client.Init(regionName)

		if err != nil {
			return "", err
		}

		// TODO 응답 시간이 3초 이상인 경우 추후 go routine 을 이용한 코드로 변경
		// // AZ 의 Instance standard 모델과 Spot 모델 조회
		standardInfo, err := describeZoneInstanceConfigInfos(client, filterMap)

		if err != nil {
			return "", err
		}

		// TODO RI 조회의 경우 tencent 는 몇가지 문제점으로 인해 추후 디벨롭하는 방향으로 제안
		// 문제점 1) client profile 의 응답 타입을 영어로 설정했지만 zone 정보가 한문으로 나온다 - 한문과 영어 zone 정보에 대한 매핑 정보 필요
		// AZ 의 RI 모델 조회
		//reservedInfo, err := describeReservedInstancesConfigInfos(client, filterMap)
		//
		//if err != nil {
		//	return "", err
		//}

		res, err := mappingToComputeStruct(filterMap, client.GetRegion(), &instanceModel{standardInfo: standardInfo /*, reservedInfo: reservedInfo*/})

		if err != nil {
			return "", err
		}

		mar, err := json.Marshal(&res)
		if err != nil {
			return "", err
		}

		return string(mar), nil
	}

	return "", nil
}

func createClientByRegionName(credential common.CredentialIface, regionPram, originalRegion string) (*cvm.Client, error) {
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"
	cpf.Language = "en-US" //메시지를 영어로 설정
	region := regionPram
	if region == "" {
		region = originalRegion
	}

	client, err := cvm.NewClient(credential, region, cpf)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func describeZoneInstanceConfigInfos(client *cvm.Client, filterMap map[string]*cvm.Filter) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {

	filters := parseToFilterSlice(filterMap, "zoneName", "instanceFamily", "instanceType")

	req := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	req.Filters = filters

	res, err := client.DescribeZoneInstanceConfigInfos(req)
	if err != nil {
		// TODO Error mapping
		return nil, err
	}

	return res, nil
}

func describeReservedInstancesConfigInfos(client *cvm.Client, filterMap map[string]*cvm.Filter) (*cvm.DescribeReservedInstancesConfigInfosResponse, error) {
	filters := parseToFilterSlice(filterMap, "zoneName")

	req := cvm.NewDescribeReservedInstancesConfigInfosRequest()
	req.Filters = filters

	res, err := client.DescribeReservedInstancesConfigInfos(req)
	if err != nil {
		// TODO Error mapping
		return nil, err
	}

	return res, nil
}

func mappingToComputeStruct(filterMap map[string]*cvm.Filter, regionName string, instanceModel *instanceModel) (*irs.CloudPriceData, error) {
	priceMap := make(map[string]*productAndPrice, 0)

	if instanceModel.standardInfo != nil {
		for _, v := range instanceModel.standardInfo.Response.InstanceTypeQuotaSet {

			// TODO filter 조건이 추가된다면 여기에: 공통으로 properties 를 필터링하는 방법 고민 필요
			key := computeInstanceKeyGeneration(v.Zone, v.InstanceType, v.CpuType)

			if pp, ok := priceMap[key]; !ok {
				sp := make([]standardVmPrice, 0)
				sp = append(sp, standardVmPrice{InstanceChargeType: v.InstanceChargeType, Price: v.Price})

				priceMap[key] = &productAndPrice{
					PriceList: &irs.PriceList{
						ProductInfo: mappingProductInfo(regionName, *v),
					},
					StandardPrices: &sp,
				}
			} else {
				newSlice := append(*pp.StandardPrices, standardVmPrice{InstanceChargeType: v.InstanceChargeType, Price: v.Price})
				pp.StandardPrices = &newSlice
			}
		}
	}

	// O(N^4) 보다 더 좋은 방법은 없을까?
	// 리프에 zone 정보가 있고 zone 별로 product 를 매핑시킨다.
	// config info 와 families 는 요소가 많지 않고 일반적으로 1~2개의 요소만을 포함하기 때문에
	// 마지막 루프가 유의미한 반복
	if instanceModel.reservedInfo != nil {
		for _, v := range instanceModel.reservedInfo.Response.ReservedInstanceConfigInfos {
			for _, info := range v.InstanceFamilies {
				for _, iType := range info.InstanceTypes {
					for _, p := range iType.Prices {

						// TODO filter 조건이 추가된다면 여기에: 공통으로 properties 를 필터링하는 방법 고민 필요
						key := computeInstanceKeyGeneration(p.Zone, iType.InstanceType, iType.CpuModelName)

						if pp, ok := priceMap[key]; !ok {
							rp := make([]reservedVmPrice, 0)
							rp = append(rp, reservedVmPrice{Price: p})

							priceMap[key] = &productAndPrice{
								PriceList: &irs.PriceList{
									ProductInfo: mappingProductInfo(regionName, *iType),
								},
								ReservedPrices: &rp,
							}
						} else {
							newSlice := append(*pp.ReservedPrices, reservedVmPrice{Price: p})
							pp.ReservedPrices = &newSlice
						}
					}

				}
			}
		}
	}

	generatePriceInfo(priceMap)

	var priceList []irs.PriceList

	if priceMap != nil && len(priceMap) > 0 {
		for _, v := range priceMap {
			priceList = append(priceList, *v.PriceList)
		}
	}

	return &irs.CloudPriceData{
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
	}, nil
}

func generatePriceInfo(priceMap map[string]*productAndPrice) {
	if priceMap != nil && len(priceMap) > 0 {
		for _, v := range priceMap {
			pl := v.PriceList
			policies := make([]irs.PricingPolicies, 0)
			prices := make([]any, 0)

			if v.StandardPrices != nil && len(*v.StandardPrices) > 0 {
				for _, val := range *v.StandardPrices {
					prices = append(prices, val.Price)
					policies = append(policies, mappingPricingPolicy(val.InstanceChargeType, *val.Price))
				}
			}

			if v.ReservedPrices != nil && len(*v.ReservedPrices) > 0 {
				for _, val := range *v.ReservedPrices {
					prices = append(prices, val.Price)
					policies = append(policies, mappingPricingPolicy(common.StringPtr("RESERVED"), *val.Price))
				}
			}

			mar, err := json.Marshal(prices)

			if err != nil {
				continue
			}

			pl.PriceInfo = irs.PriceInfo{
				PricingPolicies: policies,
				CSPPriceInfo:    string(mar),
			}
		}
	}
}

func mappingProductInfo(regionName string, i interface{}) irs.ProductInfo {
	mar, err := json.Marshal(i)

	if err != nil {
		return irs.ProductInfo{}
	}

	productInfo := irs.ProductInfo{
		ProductId:      "NA",
		RegionName:     regionName,
		CSPProductInfo: string(mar),
	}

	switch v := i.(type) {
	case cvm.InstanceTypeQuotaItem:
		vm := i.(cvm.InstanceTypeQuotaItem)

		productInfo.InstanceType = strPtrNilCheck(vm.InstanceChargeType)
		productInfo.Vcpu = intPtrNilCheck(vm.Cpu)
		productInfo.Memory = intPtrNilCheck(vm.Memory)
		productInfo.Gpu = intPtrNilCheck(vm.Gpu)
		productInfo.Description = strPtrNilCheck(vm.CpuType)

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

	case cvm.ReservedInstanceTypeItem:
		reservedVm := i.(cvm.ReservedInstanceTypeItem)

		productInfo.InstanceType = strPtrNilCheck(reservedVm.InstanceType)
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
		policyInfo.LeaseContractLength = strconv.FormatUint(*duration/31536000, 32) + "Yrs"
		policyInfo.PurchaseOption = strPtrNilCheck(p.OfferingType)

		// NA
		policyInfo.OfferingClass = strPtrNilCheck(nil)

	default:
		spew.Dump(v)
	}

	return policy
}

func computeInstanceKeyGeneration(hashingKeys ...*string) string {
	h := fnv.New32a()

	for _, key := range hashingKeys {
		if key != nil {
			_, err := h.Write([]byte(*key))
			if err != nil {
				return ""
			}
		}
	}
	return strconv.FormatUint(uint64(h.Sum32()), 10)
}

func mapToFilter(additionalFilterList []irs.KeyValue) map[string]*cvm.Filter {
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

func strPtrNilCheck(t *string) string {
	if t != nil {
		return *t
	}
	return "NA"
}

func intPtrNilCheck(t *int64) string {
	if t != nil {
		return strconv.FormatInt(*t, 32)
	}
	return "NA"
}

func uintPtrNilCheck(t *uint64) string {
	if t != nil {
		return strconv.FormatUint(*t, 32)
	}
	return "NA"
}

func floatPtrNilCheck(t *float64) string {
	if t != nil {
		return strconv.FormatFloat(*t, 'f', -1, 64)
	}
	return "NA"
}
