// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"log"

	bssopenapi "github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	/*
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaPriceInfoHandler struct {
	Region    idrv.RegionInfo
	Client    *ecs.Client
	BssClient *bssopenapi.Client
}

type ProductInfo struct {
	diskType    string
	diskMinSize int64
	diskMaxSize int64
	unit        string
}

func (priceInfoHandler *AlibabaPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	var familyList []string

	//request := ecs.CreateDescribeInstanceTypeFamiliesRequest()
	request := bssopenapi.CreateQueryProductListRequest()
	request.Scheme = "https"

	// //RegionId: tea.String("cn-hongkong"),
	request.RegionId = regionName

	//response, err := priceInfoHandler.Client.DescribeInstanceTypeFamilies(instanceFamilyRequest)
	response, err := priceInfoHandler.BssClient.QueryProductList(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	log.Println("rr")
	log.Println(response)

	// for _, instanceFamily := range response.InstanceTypeFamilies.InstanceTypeFamily {
	// 	// type InstanceTypeFamily struct {
	// 	// 	Generation           string `json:"Generation" xml:"Generation"`
	// 	// 	InstanceTypeFamilyId string `json:"InstanceTypeFamilyId" xml:"InstanceTypeFamilyId"`
	// 	// }
	// 	instanceTypeFamilyId := instanceFamily.InstanceTypeFamilyId
	// 	//generation := instanceFamily.Generation
	// 	log.Println(instanceFamily)
	// 	familyList = append(familyList, instanceTypeFamilyId)
	// }

	return familyList, nil
}

func (priceInfoHandler *AlibabaPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter []irs.KeyValue) (string, error) {
	log.Println(productFamily)
	log.Println(regionName)

	// ecs-4 ecs.smt-bw42f1d9-d
	// cn-hongkong
	var priceInfo string

	request := ecs.CreateDescribePriceRequest()
	request.Scheme = "https"

	// //RegionId: tea.String("cn-hongkong"),
	request.RegionId = priceInfoHandler.Region.Region
	request.InstanceType = productFamily

	request.RegionId = "cn-hongkong"
	request.InstanceType = "ecs-4 ecs.smt-bw42f1d9-d"

	response, err := priceInfoHandler.Client.DescribePrice(request)

	if err != nil {
		cblogger.Error(err)
		return priceInfo, err
	}
	log.Println("rr")
	log.Println(response)

	// "PriceInfo": {
	// 	"Price": {
	// 	  "OriginalPrice": 0.086,
	// 	  "ReservedInstanceHourPrice": 0,
	// 	  "DiscountPrice": 0,
	// 	  "Currency": "USD",
	// 	  "TradePrice": 0.086
	// 	},
	// 	"Rules": {
	// 	  "Rule": []
	// 	}
	//   }
	price := response.PriceInfo.Price
	log.Print(price)
	return priceInfo, nil
}
