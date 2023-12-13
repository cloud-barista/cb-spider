package resources

import (
	"encoding/json"
	"log"
	"strings"

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

// 1) aws 친화적 인터페이스 - tencent 의 pricing api 는 product family의 제공 방식이 없는 것 같아 일단 aws 리턴 타입에 맞춰 하드코딩

// 2) InquiryPriceRunInstances function 은 모든 인스턴스 타입에 대한 응답을 주지 않음
// region 정보로 zone 정보 확인
// zone 정보에 따라 InstanceType 별로 조회 필요 -> InstanceType 은 어디서 조회??
// ImageId 는 required 파라미터 => DescribeImages api 호출로 얻을 수 있음

// 3) InquiryPriceRunInstances 메서드가 과연 pricing 에 적합한 api 일까
// Instance charge type, Instance type, system disk, vpc 등 다양한 스펙에 따라 사용자가 선택한 인스턴스의 가격 정보를 얻는?

func (t *TencentPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {

	var res string

	switch {
	case strings.EqualFold("compute", productFamily):
		req := cvm.NewInquiryPriceRunInstancesRequest()

		placement := cvm.Placement{ // required
			Zone: &t.Region.Zone,
		}

		imageId := "img-pi0ii46r" // required

		// System Disk
		// var diskSize int64 = 50
		// var diskType string = "CLOUD_PREMIUM"
		// systemDisk := cvm.SystemDisk{
		// 	DiskSize: &diskSize,
		// 	DiskType: &diskType,
		// }

		// Instance Count
		// var instanceCount int64 = 1

		// Login Settings
		// var password = "password"
		// loginSetting := cvm.LoginSettings{
		// 	Password: &password,
		// }

		// Enhanced Service
		// enhancedService := cvm.EnhancedService{}

		// Internet Accessible
		// internetAccessible := cvm.InternetAccessible{}

		// Instance Charge Prepaid
		// instanceChargePrepaid := cvm.InstanceChargePrepaid{}

		// Instance Name
		// instanceName := "QCLOUD-TEST"

		// Instance Type
		// instanceType := "S5.16XLARGE256"

		// DataDisks
		// dataDisks := make([]*cvm.DataDisk, 0)
		// var diskSize int64 = 50
		// var diskType string = "CLOUD_PREMIUM"
		// dataDisks = append(dataDisks, &cvm.DataDisk{
		// 	DiskSize: &diskSize,
		// 	DiskType: &diskType,
		// })

		req.Placement = &placement
		req.ImageId = &imageId

		// req.SystemDisk = &systemDisk
		// req.InstanceCount = &instanceCount
		// req.LoginSettings = &loginSetting
		// req.EnhancedService = &enhancedService
		// req.InternetAccessible = &internetAccessible
		// req.InstanceChargePrepaid = &instanceChargePrepaid
		// req.InstanceName = &instanceName
		// req.InstanceType = &instanceType
		// req.DataDisks = dataDisks

		response, err := t.Client.InquiryPriceRunInstances(req)

		if err != nil {
			log.Fatal(err)
		}

		mar, err := json.Marshal(response.Response.Price.InstancePrice)
		if err != nil {
			return "", err
		}

		res = string(mar)

	}

	return res, nil
}
