package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"

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

func (t *TencentPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, additionalFilters []irs.KeyValue) (string, error) {
	switch {
	case strings.EqualFold("compute", productFamily):

		// TODO 기존 연결된 커넥션의 zone 정보에 의존하지 않도록 parameter 로 넘어오는 zone 정보로 데이터 매핑 필요
		// TODO zone 정보에 대해서 connection 재 연결 등의 방식으로 변경 시 확인 필요
		var filters []*cvm.Filter
		filters = append(filters, &cvm.Filter{
			Name:   common.StringPtr("zone"),
			Values: []*string{common.StringPtr(t.Region.Zone)},
		})

		begin := time.Now()

		// TODO 응답 시간이 3초 이상인 경우 추후 go routine 을 이용한 코드로 변경
		// AZ 의 Instance 모델과 Spot 모델 조회
		standardInfo, err := t.describeZoneInstanceConfigInfos(filters, additionalFilters)

		if err != nil {
			return "", err
		}

		fmt.Printf("[Timer] After describe zone instance api call %s\n", time.Since(begin).Truncate(time.Millisecond))

		// AZ 의 RI 모델 조회
		reservedInfo, err := t.describeReservedInstancesConfigInfos(filters, additionalFilters)

		if err != nil {
			return "", err
		}

		fmt.Printf("[Timer] After describe reserved instance api call %s\n", time.Since(begin).Truncate(time.Millisecond))

		// res, err := mappingToStruct(regionName, &instanceModel{standardInfo: standardInfo, reservedInfo: reservedInfo})
		res, err := mappingToStruct(regionName, &instanceModel{standardInfo: standardInfo, reservedInfo: reservedInfo})

		if err != nil {
			return "", err
		}

		fmt.Printf("[Timer] After mapping struct %s\n", time.Since(begin).Truncate(time.Millisecond))

		mar, err := json.MarshalIndent(&res, "", "  ")

		fmt.Printf("[Timer] After marsharling %s\n", time.Since(begin).Truncate(time.Millisecond))

		if err != nil {
			return "", err
		}

		return string(mar), nil
	}

	return "", nil
}

func (t *TencentPriceInfoHandler) describeZoneInstanceConfigInfos(filters []*cvm.Filter, filterList []irs.KeyValue) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {

	filters = append(filters, &cvm.Filter{
		Name:   common.StringPtr("dtatus"),
		Values: []*string{common.StringPtr("SELL")},
	})

	for _, kv := range filterList {
		switch kv.Key {
		case "instance-family":
			filters = append(filters, &cvm.Filter{
				Name:   common.StringPtr("instance-family"),
				Values: []*string{common.StringPtr(kv.Value)},
			})
		case "instance-type":
			filters = append(filters, &cvm.Filter{
				Name:   common.StringPtr("instance-type"),
				Values: []*string{common.StringPtr(kv.Value)},
			})
		default:
		}
	}

	req := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	req.Filters = filters

	res, err := t.Client.DescribeZoneInstanceConfigInfos(req)
	if err != nil {
		// TODO Error mapping
		return nil, err
	}

	return res, nil
}

func (t *TencentPriceInfoHandler) describeReservedInstancesConfigInfos(filters []*cvm.Filter, filterList []irs.KeyValue) (*cvm.DescribeReservedInstancesConfigInfosResponse, error) {

	req := cvm.NewDescribeReservedInstancesConfigInfosRequest()
	req.Filters = filters

	res, err := t.Client.DescribeReservedInstancesConfigInfos(req)
	if err != nil {
		// TODO Error mapping
		return nil, err
	}

	return res, nil
}

func mappingToStructTogether(regionName string, instanceModel *instanceModel) (*irs.CloudPriceData, error) {

	// map 객체에 저장해보자.
	reservedMap := make(map[string]*cvm.ReservedInstanceTypeItem)

	if instanceModel.reservedInfo != nil {
		for _, v := range instanceModel.reservedInfo.Response.ReservedInstanceConfigInfos {
			for _, info := range v.InstanceFamilies {
				for _, iType := range info.InstanceTypes {
					reservedMap[*iType.InstanceType] = iType
				}
			}
		}
	}

	var priceList []irs.PriceList

	if instanceModel.standardInfo != nil {
		for _, v := range instanceModel.standardInfo.Response.InstanceTypeQuotaSet {

			priceList = append(priceList, irs.PriceList{
				ProductInfo: mappindgStandardProductInfo(regionName, v),
				PriceInfo:   mappingStandardPriceInfoWithReserved(v, reservedMap[*v.InstanceType]),
			})
		}
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

func mappingToStruct(regionName string, instanceModel *instanceModel) (*irs.CloudPriceData, error) {
	var priceList []irs.PriceList

	if instanceModel.standardInfo != nil {
		for _, v := range instanceModel.standardInfo.Response.InstanceTypeQuotaSet {
			priceList = append(priceList, irs.PriceList{
				ProductInfo: mappindgStandardProductInfo(regionName, v),
				PriceInfo:   mappingStandardPriceInfo(v),
			})
		}
	}

	// O(N^3) 보다 더 좋은 방법은??
	// config info 와 families 는 요소가 많지 않고 보통 1~2개의 요소만을 포함하기 때문에
	// 마지막 루프가 유의미한 반복인 확률이 가장 높음.
	// O(N^3)의 시간복잡도를 가지지만 오랜 시간이 걸리지 않을 것으로 판단.
	if instanceModel.reservedInfo != nil {
		for _, v := range instanceModel.reservedInfo.Response.ReservedInstanceConfigInfos {
			for _, info := range v.InstanceFamilies {
				for _, iType := range info.InstanceTypes {
					priceList = append(priceList, irs.PriceList{
						ProductInfo: mappingReservedProductInfo(regionName, iType),
						PriceInfo:   mappingReservedPriceInfo(iType),
					})
				}
			}
		}
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

func mappindgStandardProductInfo(regionName string, standardItem *cvm.InstanceTypeQuotaItem) irs.ProductInfo {
	mar, err := json.MarshalIndent(standardItem, "", "  ")

	if err != nil {
		return irs.ProductInfo{}
	}

	productInfo := irs.ProductInfo{
		ProductId:  "I don't know what to fill",
		RegionName: regionName,

		InstanceType:   *standardItem.InstanceType,
		Vcpu:           strconv.FormatInt(*standardItem.Cpu, 32),
		Memory:         strconv.FormatInt(*standardItem.Memory, 32),
		Gpu:            strconv.FormatInt(*standardItem.Gpu, 32),
		Description:    *standardItem.CpuType,
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
		ProductId:  "",
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

func mappingStandardPriceInfoWithReserved(standard *cvm.InstanceTypeQuotaItem, reserved *cvm.ReservedInstanceTypeItem) irs.PriceInfo {
	price := standard.Price
	var prices []*cvm.ReservedInstancePriceItem
	if reserved != nil {
		prices = reserved.Prices
	}

	mar, err := json.MarshalIndent(map[string]any{"standard": price, "reserved": prices}, "", "  ")
	if err != nil {
		return irs.PriceInfo{}
	}

	var policies []irs.PricingPolicies

	// standard pricing info mapping
	policyInfo := irs.PricingPolicyInfo{
		LeaseContractLength: "",
		OfferingClass:       "",
		PurchaseOption:      "",
	}

	policy := irs.PricingPolicies{
		PricingPolicy:     *standard.InstanceChargeType,
		Unit:              *price.ChargeUnit,
		Currency:          "USD",
		Price:             strconv.FormatFloat(*price.UnitPrice, 'f', -1, 64),
		PricingPolicyInfo: &policyInfo,
	}

	policies = append(policies, policy)

	if len(prices) > 0 {
		for _, price := range prices {
			policyInfo := irs.PricingPolicyInfo{
				LeaseContractLength: strconv.FormatUint(*price.Duration/31536000, 32),
				OfferingClass:       "",
				PurchaseOption:      *price.OfferingType,
			}

			policy := irs.PricingPolicies{
				PricingId:         *price.ReservedInstancesOfferingId,
				PricingPolicy:     "Reserved",
				Unit:              "yrs",
				Currency:          "USD",
				Price:             strconv.FormatFloat(*price.FixedPrice, 'f', -1, 64),
				PricingPolicyInfo: &policyInfo,
				Description:       *price.ProductDescription,
			}

			policies = append(policies, policy)
		}
	}

	priceInfo := irs.PriceInfo{
		PricingPolicies: policies,
		CSPPriceInfo:    string(mar),
	}

	return priceInfo
}

func mappingStandardPriceInfo(item *cvm.InstanceTypeQuotaItem) irs.PriceInfo {
	price := item.Price

	mar, err := json.MarshalIndent(price, "", "  ")
	if err != nil {
		return irs.PriceInfo{}
	}

	// price info mapping
	policyInfo := irs.PricingPolicyInfo{
		LeaseContractLength: "",
		OfferingClass:       "",
		PurchaseOption:      "",
	}

	policy := irs.PricingPolicies{
		PricingPolicy:     *item.InstanceChargeType,
		Unit:              *price.ChargeUnit,
		Currency:          "USD",
		Price:             strconv.FormatFloat(*price.UnitPrice, 'f', -1, 64),
		PricingPolicyInfo: &policyInfo,
	}

	priceInfo := irs.PriceInfo{
		PricingPolicies: []irs.PricingPolicies{policy},
		CSPPriceInfo:    string(mar),
	}

	return priceInfo
}

func mappingReservedPriceInfo(item *cvm.ReservedInstanceTypeItem) irs.PriceInfo {
	prices := item.Prices

	mar, err := json.MarshalIndent(prices, "", "  ")
	if err != nil {
		return irs.PriceInfo{}
	}

	var policies []irs.PricingPolicies

	for _, price := range prices {
		policyInfo := irs.PricingPolicyInfo{
			LeaseContractLength: strconv.FormatUint(*price.Duration/31536000, 32),
			OfferingClass:       "",
			PurchaseOption:      *price.OfferingType,
		}

		policy := irs.PricingPolicies{
			PricingId:         *price.ReservedInstancesOfferingId,
			PricingPolicy:     "Reserved",
			Unit:              "yrs",
			Currency:          "USD",
			Price:             strconv.FormatFloat(*price.FixedPrice, 'f', -1, 64),
			PricingPolicyInfo: &policyInfo,
			Description:       *price.ProductDescription,
		}

		policies = append(policies, policy)
	}

	priceInfo := irs.PriceInfo{
		PricingPolicies: policies,
		CSPPriceInfo:    string(mar),
	}

	return priceInfo
}

// func (t *TencentPriceInfoHandler) vmInquiryPriceRunInstacne(filterList []irs.KeyValue) string {
// 	req := cvm.NewInquiryPriceRunInstancesRequest()

// 	placement := cvm.Placement{ // required
// 		Zone: common.StringPtr(t.Region.Zone),
// 	}

// 	imageId, err := getFiltereValue(filterList, "imageId") // required -> 이미지 Id 는 사용자에게서 받아와야 함

// 	if err != nil {
// 		return ""
// 	}

// 	// System Disk
// 	// var diskSize int64 = 50
// 	// var diskType string = "CLOUD_PREMIUM"
// 	// systemDisk := cvm.SystemDisk{
// 	// 	DiskSize: &diskSize,
// 	// 	DiskType: &diskType,
// 	// }

// 	// Instance Count
// 	// var instanceCount int64 = 1

// 	// Login Settings
// 	// var password = "password"
// 	// loginSetting := cvm.LoginSettings{
// 	// 	Password: &password,
// 	// }

// 	// Enhanced Service
// 	// enhancedService := cvm.EnhancedService{}

// 	// Internet Accessible
// 	// internetAccessible := cvm.InternetAccessible{}

// 	// Instance Charge Prepaid
// 	// instanceChargePrepaid := cvm.InstanceChargePrepaid{}

// 	// Instance Name
// 	// instanceName := "QCLOUD-TEST"

// 	// Instance Type
// 	// instanceType := "S5.16XLARGE256"

// 	// DataDisks
// 	// dataDisks := make([]*cvm.DataDisk, 0)
// 	// var diskSize int64 = 50
// 	// var diskType string = "CLOUD_PREMIUM"
// 	// dataDisks = append(dataDisks, &cvm.DataDisk{
// 	// 	DiskSize: &diskSize,
// 	// 	DiskType: &diskType,
// 	// })

// 	req.Placement = &placement
// 	req.ImageId = &imageId

// 	// req.SystemDisk = &systemDisk
// 	// req.InstanceCount = &instanceCount
// 	// req.LoginSettings = &loginSetting
// 	// req.EnhancedService = &enhancedService
// 	// req.InternetAccessible = &internetAccessible
// 	// req.InstanceChargePrepaid = &instanceChargePrepaid
// 	// req.InstanceName = &instanceName
// 	// req.InstanceType = &instanceType
// 	// req.DataDisks = dataDisks

// 	response, err := t.Client.InquiryPriceRunInstances(req)
// 	t.Client.DescribeZoneInstanceConfigInfos()

// 	t.Client.DescribeInstace

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	mar, err := json.Marshal(response.Response.Price)
// 	if err != nil {
// 		return "", err
// 	}

// 	res = string(mar)

// 	return res
// }

func getFiltereValue(filterList []irs.KeyValue, key string) (string, error) {
	for _, kv := range filterList {
		if strings.EqualFold(key, kv.Key) {
			return kv.Value, nil
		}
	}
	return "", errors.New("No exist key")
}
