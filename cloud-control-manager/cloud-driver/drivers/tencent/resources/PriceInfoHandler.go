package resources

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
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

func (t *TencentPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	switch {
	case strings.EqualFold("compute", productFamily):

		priceInfo, err := t.describeZoneInstanceConfigInfos(t.Region.Zone, filterList)

		if err != nil {
			return "", err
		}

		res, err := mappingToStruct(priceInfo, regionName, filterList)

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

func (t *TencentPriceInfoHandler) describeZoneInstanceConfigInfos(zone string, filterList []irs.KeyValue) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
	filters := []*cvm.Filter{
		{
			Name:   common.StringPtr("zone"),
			Values: []*string{common.StringPtr(zone)},
		},
		{
			Name:   common.StringPtr("instance-family"),
			Values: []*string{common.StringPtr("SN3ne")},
		},
		{
			Name:   common.StringPtr("instance-type"),
			Values: []*string{common.StringPtr("SN3ne.SMALL2")},
		},
	}

	req := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	req.Filters = filters

	res, err := t.Client.DescribeZoneInstanceConfigInfos(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func mappingToStruct(priceInfo *cvm.DescribeZoneInstanceConfigInfosResponse, regionName string, filterList []irs.KeyValue) (*irs.CloudPriceData, error) {

	var priceList []irs.PriceList

	for _, v := range priceInfo.Response.InstanceTypeQuotaSet {
		spew.Dump(v)
		priceList = append(priceList, irs.PriceList{
			ProductInfo: mappingProductInfo(v, regionName, filterList),
			PriceInfo:   mappingPriceInfo(v, regionName, filterList),
		})
	}

	return &irs.CloudPriceData{
		Meta: irs.Meta{
			Version:     "this is version",
			Description: "Get price info",
		},
		CloudPriceList: []irs.CloudPrice{
			{
				CloudName: "TENCENT",
				PriceList: priceList,
			},
		},
	}, nil
}

func mappingProductInfo(item *cvm.InstanceTypeQuotaItem, regionName string, filterList []irs.KeyValue) irs.ProductInfo {
	mar, err := json.MarshalIndent(item, "", "  ")

	if err != nil {
		return irs.ProductInfo{}
	}

	productInfo := irs.ProductInfo{
		ProductId:  "",
		RegionName: regionName,

		InstanceType:   *item.InstanceType,
		Vcpu:           strconv.FormatInt(*item.Cpu, 32),
		Memory:         strconv.FormatInt(*item.Memory, 32),
		Gpu:            strconv.FormatInt(*item.Gpu, 32),
		CSPProductInfo: string(mar),
	}
	return productInfo
}

func mappingPriceInfo(item *cvm.InstanceTypeQuotaItem, regionName string, filterList []irs.KeyValue) irs.PriceInfo {
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
