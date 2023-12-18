// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	bssopenapi "github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	/*
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaPriceInfoHandler struct {
	// Region idrv.RegionInfo
	// Client    *ecs.Client
	BssClient *bssopenapi.Client
}

type ProductInfo struct {
	diskType    string
	diskMinSize int64
	diskMaxSize int64
	unit        string
}

func (priceInfoHandler *AlibabaPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	// https://api.alibabacloud.com/document/BssOpenApi/2017-12-14/DescribeResourcePackageProduct?spm=api-workbench-intl.api_explorer.0.0.777f813524Q25K
	// API docs 상 가능 Region 명시되지 않음, 티켓에서도 별도 안내 없음.
	// 모든 Region 테스트 결과 아래 6개 리전에서 bss API 권한을 가진 상태로 정상 결과 응답
	// ++ QueryProductList 는 클라이언트에서 Region 정보를 가져오는 것이 아닌, 별도 Input 으로 리전 받음
	// ++ 제공되는 Product 는 23.12.18 현재 123개로 모든 리전에서 동일한 응답
	// Tested request Region
	// us-east-1, us-west-1, eu-west-1, eu-central-1, ap-south-1, me-east-1,

	pricingRegion := []string{"us-east-1", "us-west-1", "eu-west-1", "eu-central-1", "ap-south-1", "me-east-1"} // updated : 23.12.18
	match := false
	for _, str := range pricingRegion {
		if str == regionName {
			match = true
			break
		}
	}

	var targetRegion string
	if match {
		targetRegion = regionName
	} else {
		targetRegion = "us-east-1"
	}

	request := bssopenapi.CreateQueryProductListRequest()
	request.Scheme = "https"
	request.RegionId = targetRegion
	request.QueryTotalCount = requests.Boolean("true")
	request.PageNum = requests.NewInteger(1)
	request.PageSize = requests.NewInteger(1)

	// 전체 서비스 카운트를 얻어오기 위해 PageNum과 PageSize를 1로 설정하여 QueryTotalCount 획득
	responseTotalcount, err := priceInfoHandler.BssClient.QueryProductList(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	// QueryTotalCount 설정. 23.12.18 : 123개
	request.PageSize = requests.NewInteger(responseTotalcount.Data.TotalCount)
	productListresponse, err := priceInfoHandler.BssClient.QueryProductList(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var familyList []string
	for _, Product := range productListresponse.Data.ProductList.Product {
		familyList = append(familyList, Product.ProductCode)
	}

	return familyList, nil
}

func (priceInfoHandler *AlibabaPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter []irs.KeyValue) (string, error) {
	// log.Println(productFamily)
	// log.Println(regionName)

	// // ecs-4 ecs.smt-bw42f1d9-d
	// // cn-hongkong
	// var priceInfo string

	// request := ecs.CreateDescribePriceRequest()
	// request.Scheme = "https"

	// // //RegionId: tea.String("cn-hongkong"),
	// request.RegionId = priceInfoHandler.Region.Region
	// request.InstanceType = productFamily

	// request.RegionId = "cn-hongkong"
	// request.InstanceType = "ecs-4 ecs.smt-bw42f1d9-d"

	// response, err := priceInfoHandler.Client.DescribePrice(request)

	// if err != nil {
	// 	cblogger.Error(err)
	// 	return priceInfo, err
	// }
	// log.Println("rr")
	// log.Println(response)

	// // "PriceInfo": {
	// // 	"Price": {
	// // 	  "OriginalPrice": 0.086,
	// // 	  "ReservedInstanceHourPrice": 0,
	// // 	  "DiscountPrice": 0,
	// // 	  "Currency": "USD",
	// // 	  "TradePrice": 0.086
	// // 	},
	// // 	"Rules": {
	// // 	  "Rule": []
	// // 	}
	// //   }
	// price := response.PriceInfo.Price
	// log.Print(price)
	return "priceInfo", nil
}
