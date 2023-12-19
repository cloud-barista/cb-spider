package resources

import (
	"encoding/json"
	"hash/fnv"
	"strconv"
	"strings"

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

// 제공되는 product family list 를 가져오는 api 를 찾을 수 없음...
// 이런 경우 어떤 방식으로 인터페이스를 처리하는거지?
func (t *TencentPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	pl := make([]string, 0)
	return pl, nil
}

type instanceModel struct {
	standardInfo *cvm.DescribeZoneInstanceConfigInfosResponse
	reservedInfo *cvm.DescribeReservedInstancesConfigInfosResponse
}

type productAndPrice struct {
	PriceList      *irs.PriceList
	StandardPrices *[]standardPrice
	ReservedPrices *[]reservedPrice
}

type standardPrice struct {
	InstanceChargeType *string
	Price              *cvm.ItemPrice
}

type reservedPrice struct {
	Price *cvm.ReservedInstancePriceItem
}

func (t *TencentPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, additionalFilters []irs.KeyValue) (string, error) {

	filterMap := mapToFilter(additionalFilters)

	switch {
	case strings.EqualFold("compute", productFamily):

		// client 생성 with zone
		client, err := createClientByRegionName(t.Client.GetCredential(), regionName, t.Region.Region)

		if err != nil {
			return "", err
		}

		// TODO 응답 시간이 3초 이상인 경우 추후 go routine 을 이용한 코드로 변경
		// // AZ 의 Instance standard 모델과 Spot 모델 조회
		standardInfo, err := describeZoneInstanceConfigInfos(client, filterMap)

		if err != nil {
			return "", err
		}

		// TODO RI 조회의 경우 tencent 는 몇가지 문제점으로 인해 추후 디벨롭하는 방향으로 제안해보면 어떨까
		// 문제점 1) client profile 의 응답 타입을 영어로 설정했지만 zone 정보가 한문으로 나온다 - 한문과 영어 zone 정보에 대한 매핑 정보 필요
		// AZ 의 RI 모델 조회
		// reservedInfo, err := describeReservedInstancesConfigInfos(client, filterMap)

		// if err != nil {
		// 	return "", err
		// }

		res, err := mappingToStruct(filterMap, client.GetRegion(), &instanceModel{standardInfo: standardInfo /* , reservedInfo: reservedInfo */ })

		if err != nil {
			return "", err
		}

		mar, err := json.MarshalIndent(&res, "", "  ")
		if err != nil {
			return "", err
		}

		return string(mar), nil
	}

	return "", nil
}

func createClientByRegionName(credentialIface common.CredentialIface, regionPram, originalRegion string) (*cvm.Client, error) {
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"
	cpf.Language = "en-US" //메시지를 영어로 설정
	region := regionPram
	if region == "" {
		region = originalRegion
	}

	client, err := cvm.NewClient(credentialIface, region, cpf)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func describeZoneInstanceConfigInfos(client *cvm.Client, filterMap map[string]*cvm.Filter) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {

	filters := parseToFilterSlice(filterMap, "zoneName", "instance-family", "instance-type")

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

func mappingToStruct(filterMap map[string]*cvm.Filter, regionName string, instanceModel *instanceModel) (*irs.CloudPriceData, error) {
	// zone 과 instance type 으로 hashing
	// zone-instance-type 에 대해 하위 price policy -> pay-as-you-go, spot, reserved 가격 정보 필요
	priceMap := make(map[string]*productAndPrice, 0)

	if instanceModel.standardInfo != nil {
		for _, v := range instanceModel.standardInfo.Response.InstanceTypeQuotaSet {

			key := keyGeneration(v.Zone, v.InstanceType, v.CpuType, v.Cpu, v.Memory)

			if pp, ok := priceMap[key]; !ok {
				sp := make([]standardPrice, 0)
				sp = append(sp, standardPrice{InstanceChargeType: v.InstanceChargeType, Price: v.Price})

				priceMap[key] = &productAndPrice{
					PriceList: &irs.PriceList{
						ProductInfo: mappingProductInfo(regionName, v),
					},
					StandardPrices: &sp,
				}
			} else {
				newSlice := append(*pp.StandardPrices, standardPrice{InstanceChargeType: v.InstanceChargeType, Price: v.Price})
				pp.StandardPrices = &newSlice
			}
		}
	}

	// O(N^4) 보다 더 좋은 방법은?? -> 최하단 뎁스에 zone 정보가 있고 zone 별로 product 를 매핑시킨다.
	// config info 와 families 는 요소가 많지 않고 보통 1~2개의 요소만을 포함하기 때문에
	// 마지막 루프가 유의미한 반복인 확률이 가장 높음.
	if instanceModel.reservedInfo != nil {
		for _, v := range instanceModel.reservedInfo.Response.ReservedInstanceConfigInfos {
			for _, info := range v.InstanceFamilies {
				for _, iType := range info.InstanceTypes {
					for _, p := range iType.Prices {

						// TODO iType.InstanceType 과 filterMap의 instance-type 과 비교 필요
						key := keyGeneration(p.Zone, iType.InstanceType, iType.CpuModelName, convertUnsignedToSignedPointer64(iType.Cpu), convertUnsignedToSignedPointer64(iType.Memory))

						if pp, ok := priceMap[key]; !ok {
							rp := make([]reservedPrice, 0)
							rp = append(rp, reservedPrice{Price: p})

							priceMap[key] = &productAndPrice{
								PriceList: &irs.PriceList{
									ProductInfo: mappingReservedProductInfo(regionName, iType),
								},
								ReservedPrices: &rp,
							}
						} else {
							newSlice := append(*pp.ReservedPrices, reservedPrice{Price: p})
							pp.ReservedPrices = &newSlice
						}
					}

				}
			}
		}
	}

	generatePriceInfo(priceMap)

	var priceList []irs.PriceList

	for _, v := range priceMap {
		priceList = append(priceList, *v.PriceList)
	}

	return &irs.CloudPriceData{
		Meta: irs.Meta{
			Version:     "This is version info.",
			Description: "This is description of this function.",
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
	for _, v := range priceMap {
		pl := v.PriceList

		if v.StandardPrices != nil && len(*v.StandardPrices) > 0 {
			policies := make([]irs.PricingPolicies, 0)
			prices := make([]*cvm.ItemPrice, 0)
			for _, val := range *v.StandardPrices {
				prices = append(prices, val.Price)
				policies = append(policies, mappingPricePolicy(val.InstanceChargeType, val.Price))
			}

			mar, err := json.MarshalIndent(prices, "", "  ")

			if err != nil {
				continue
			}

			pl.PriceInfo = irs.PriceInfo{
				PricingPolicies: policies,
				CSPPriceInfo:    string(mar),
			}
		}

		if v.ReservedPrices != nil && len(*v.ReservedPrices) > 0 {
			policies := make([]irs.PricingPolicies, 0)
			prices := make([]*cvm.ReservedInstancePriceItem, 0)

			for _, val := range *v.ReservedPrices {
				prices = append(prices, val.Price)
				policies = append(policies, mappingReservedPriceInfo(val.Price))
			}

			mar, err := json.MarshalIndent(prices, "", "  ")

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

func mappingProductInfo(regionName string, item *cvm.InstanceTypeQuotaItem) irs.ProductInfo {
	mar, err := json.MarshalIndent(item, "", "  ")

	if err != nil {
		return irs.ProductInfo{}
	}

	productInfo := irs.ProductInfo{
		ProductId:  "NA",
		RegionName: regionName,

		InstanceType:   *item.InstanceType,
		Vcpu:           strconv.FormatInt(*item.Cpu, 32),
		Memory:         strconv.FormatInt(*item.Memory, 32),
		Gpu:            strconv.FormatInt(*item.Gpu, 32),
		Description:    *item.CpuType,
		CSPProductInfo: string(mar),
	}
	return productInfo
}

func mappingReservedProductInfo(regionName string, reservedItem *cvm.ReservedInstanceTypeItem) irs.ProductInfo {
	mar, err := json.MarshalIndent(reservedItem, "", "  ")

	if err != nil {
		return irs.ProductInfo{}
	}

	productInfo := irs.ProductInfo{
		ProductId:  "NA",
		RegionName: regionName,

		InstanceType:   *reservedItem.InstanceType,
		Vcpu:           strconv.FormatUint(*reservedItem.Cpu, 32),
		Memory:         strconv.FormatUint(*reservedItem.Memory, 32),
		Gpu:            strconv.FormatUint(*reservedItem.Gpu, 32),
		Description:    *reservedItem.CpuModelName,
		CSPProductInfo: string(mar),
	}
	return productInfo
}

func mappingPricePolicy(instanceChargeType *string, price *cvm.ItemPrice) irs.PricingPolicies {
	// price info mapping
	policyInfo := irs.PricingPolicyInfo{
		LeaseContractLength: "",
		OfferingClass:       "",
		PurchaseOption:      "",
	}

	policy := irs.PricingPolicies{
		PricingId:         "NA",
		PricingPolicy:     *instanceChargeType,
		Unit:              *price.ChargeUnit,
		Currency:          "USD",
		Price:             strconv.FormatFloat(*price.UnitPrice, 'f', -1, 64),
		PricingPolicyInfo: &policyInfo,
	}

	return policy
}

func mappingReservedPriceInfo(reservedPrice *cvm.ReservedInstancePriceItem) irs.PricingPolicies {

	policyInfo := irs.PricingPolicyInfo{
		LeaseContractLength: strconv.FormatUint(*reservedPrice.Duration/31536000, 32),
		OfferingClass:       "",
		PurchaseOption:      *reservedPrice.OfferingType,
	}

	policy := irs.PricingPolicies{
		PricingId:         *reservedPrice.ReservedInstancesOfferingId,
		PricingPolicy:     "Reserved",
		Unit:              "yrs",
		Currency:          "USD",
		Price:             strconv.FormatFloat(*reservedPrice.FixedPrice, 'f', -1, 64),
		PricingPolicyInfo: &policyInfo,
		Description:       *reservedPrice.ProductDescription,
	}

	return policy
}

func keyGeneration(zone, instanceType, cpuType *string, cpu, memory *int64) string {
	h := fnv.New32a()
	h.Write([]byte(*zone))
	h.Write([]byte(*instanceType))
	h.Write([]byte(*cpuType))
	h.Write([]byte(strconv.FormatInt(*cpu, 10)))
	h.Write([]byte(strconv.FormatInt(*memory, 10)))
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

func convertUnsignedToSignedPointer64(p *uint64) *int64 {
	if p == nil {
		return nil
	}
	x := int64(*p)
	return &x
}
