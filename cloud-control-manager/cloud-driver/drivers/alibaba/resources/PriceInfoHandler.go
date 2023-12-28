// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"fmt"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	bssopenapi "github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi" // update to v1.62.327 from v1.61.1743, due to QuerySkuPriceListRequest struct
	"github.com/davecgh/go-spew/spew"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

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
	// ++ QueryProductList 는 클라이언트에서 Region 정보를 가져오는 것이 아닌, 별도 Input 으로 리전 받음, 따라서 별도 Client 에 리전 셋팅 필요 없음.
	// ++ 제공되는 Product 는 23.12.18 현재 123개로 모든 리전에서 동일한 응답을 확인
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
	request.QueryTotalCount = requests.Boolean("true") // 전체 서비스 카운트 리턴 옵션
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
	// QueryProductList (ProductCode) ->
	// (ProductCode) QueryCommodityListView (CommodityCode) ->
	// (CommodityCode) QueryPriceEntityListView (PriceFactorCode,PriceFactorValueList , PriceEntityCode)
	// (CommodityCode, PriceFactorCode, PriceFactorValueList, PriceEntityCode) QuerySkuPriceListViewPagination

	querySkuPriceListRequestVar := bssopenapi.CreateQuerySkuPriceListRequest()
	querySkuPriceListRequestVar.PageSize = requests.NewInteger(20)
	querySkuPriceListRequestVar.Scheme = "https"
	for _, keyValue := range filter {
		if keyValue.Key == "CommodityCode" {
			querySkuPriceListRequestVar.CommodityCode = keyValue.Value
		} else if keyValue.Key == "PriceEntityCode" {
			querySkuPriceListRequestVar.PriceEntityCode = keyValue.Value
		} else {
			values := strings.Split(keyValue.Value, ",")
			querySkuPriceListRequestVar.PriceFactorConditionMap = make(map[string]*[]string) // for nil pointer err fix
			querySkuPriceListRequestVar.PriceFactorConditionMap[keyValue.Key] = &values
			// querySkuPriceListRequestVar.PriceFactorConditionMap = bssopenapi.QuerySkuPriceListPriceFactorConditionMap{
			// 	Systemdisk_category: &[]string{"ephemeral_ssd", "cloud_ssd"},
			// }

		}
	}

	fmt.Println(querySkuPriceListRequestVar)
	spew.Dump(querySkuPriceListRequestVar)

	// response, err := priceInfoHandler.BssClient.QuerySkuPriceList(querySkuPriceListRequestVar)
	// if err != nil {
	// 	cblogger.Error(err)
	// }
	var priceList []bssopenapi.SkuPricePageDTO
	for {
		response, err := priceInfoHandler.BssClient.QuerySkuPriceList(querySkuPriceListRequestVar)
		if err != nil {
			cblogger.Error(err)
		}
		for _, price := range response.Data.SkuPricePage.SkuPriceList {
			priceList = append(priceList, price)
		}
		if response.Data.SkuPricePage.NextPageToken != "" {
			querySkuPriceListRequestVar.NextPageToken = response.Data.SkuPricePage.NextPageToken
		} else {
			break
		}

		break
	}

	fmt.Printf("response is %#v\n", priceList)

	return "priceInfo", nil
}
