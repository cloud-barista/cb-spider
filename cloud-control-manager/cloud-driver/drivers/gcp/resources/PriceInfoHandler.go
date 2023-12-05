package resources

import (
	"context"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"

	"google.golang.org/api/cloudbilling/v1"
	compute "google.golang.org/api/compute/v1"
)

// API를 호출하는 데 특정 IAM 권한이 필요하지 않습니다.
// https://cloudbilling.googleapis.com/v2beta/services?key=API_KEY&pageSize=PAGE_SIZE&pageToken=PAGE_TOKEN

type GCPPriceInfoHandler struct {
	Region             idrv.RegionInfo
	Ctx                context.Context
	Client             *compute.Service
	CloudBillingClient *cloudbilling.APIService
	//Client *cloudbilling.Service
	//Client     *billing.CloudBillingClient
	Credential idrv.CredentialInfo
}

// 해당 Region의 PriceFamily에 해당하는 제품들의 가격정보를 json형태로 return
func (priceInfoHandler *GCPPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter []irs.KeyValue) (string, error) {
	returnJson := ""
	// projectID := priceInfoHandler.Credential.ProjectID

	// resp, err := GetRegion(priceInfoHandler.Client, projectID, regionName)
	// if err != nil {
	// 	cblogger.Error(err)
	// 	return returnJson, err
	// }
	// cblogger.Debug(resp)

	return returnJson, nil
}

// product family의 이름들을 배열로 return
func (priceInfoHandler *GCPPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	returnProductFamilyNames := []string{}

	//projectID := priceInfoHandler.Credential.ProjectID

	//billingAccounts/01B26B-B0CA80-2EFE4E

	resp, err := priceInfoHandler.CloudBillingClient.Services.Skus.List("billingAccounts/01B26B-B0CA80-2EFE4E").Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	// Compute Engine SKU 및 가격 정보 가져오기

	// VM의 경우 아래 항목에 대해 가격이 매겨짐.
	// VM 인스턴스 가격 책정
	// 네트워킹 가격 책정
	// 단독 테넌트 노드 가격 책정
	// GPU 가격 책정
	// 디스크 및 이미지 가격 책정
	spew.Dump(resp)

	return returnProductFamilyNames, nil
}
