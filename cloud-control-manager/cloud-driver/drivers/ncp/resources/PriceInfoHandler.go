// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2023.12.

package resources

import (
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/hmac" // Caution!! : Not "crypto/hmac"

	// Ref) https://github.com/NaverCloudPlatform/ncloud-sdk-go-v2/blob/master/services/vserver/api_client.go

	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpPriceInfoHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *server.APIClient
}

// Already declared in CommonNcpFunc.go
// var cblogger *logrus.Logger
func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Classic VMHandler")
}

// =========================== For ProductList ============================
type ProductItemKind struct {
	Code     string `json:"code"`
	CodeName string `json:"code_name"`
}

type ProductItemKindDetail struct {
	Code     string `json:"code"`
	CodeName string `json:"code_name"`
}

type Product struct {
	ItemKind       ProductItemKind        `json:"productItemKind"`
	ItemKindDetail ProductItemKindDetail  `json:"productItemKindDetail"`
	ProductCode    string                 `json:"productCode"`
	ProductName    string                 `json:"productName"`
	Description    string                 `json:"productDescription"`
	softwareType   map[string]interface{} `json:"softwareType"`
}

type ProductListAPIResponse struct {
	GetProductListResponse struct {
		RequestID     string    `json:"request_id"`
		ReturnCode    int       `json:"return_code"`
		ReturnMessage string    `returnMessage`
		TotalRows     int       `json:"totalRows"`
		ProductList   []Product `json:"productList"`
	} `json:"getProductListResponse"`

	Error *ErrorResponse `json:"error,omitempty"`
}

// =========================== For ProductList ============================

// =========================== For PriceList ============================
type Code struct {
	Code     string `json:"code"`
	CodeName string `json:"codeName"`
}

type Region struct {
	RegionNo   int    `json:"regionNo"` // Not string
	RegionCode string `json:"regionCode"`
	RegionName string `json:"regionName"`
}

type Price struct {
	PriceNo                string  `json:"priceNo"` // Not int
	PriceType              Code    `json:"priceType"`
	Region                 Region  `json:"region"`
	ChargingUnitType       Code    `json:"chargingUnitType"`
	RatingUnitType         Code    `json:"ratingUnitType"`
	ChargingUnitBasicValue string  `json:"chargingUnitBasicValue"` // Not int
	Unit                   Code    `json:"unit"`
	PriceValue             float32 `json:"price"` // Not float64
	ConditionType          Code    `json:"conditionType"`
	ConditionPrice         float32 `json:"conditionPrice"` // Not float64
	PriceDescription       string  `json:"priceDescription"`
	MeteringUnit           Code    `json:"meteringUnit"`
	StartDate              string  `json:"startDate"`
	PayCurrency            Code    `json:"payCurrency"`
}

type ProductPrice struct {
	ProductItemKind       Code   `json:"productItemKind"`
	ProductItemKindDetail Code   `json:"productItemKindDetail"`
	ProductCode           string `json:"productCode"`
	ProductName           string `json:"productName"`
	ProductDescription    string `json:"productDescription"`
	ProductType           Code   `json:"productType"`
	GpuCount              int    `json:"gpuCount"`
	CpuCount              int    `json:"cpuCount"`
	MemorySize            int64  `json:"memorySize"`
	BaseBlockStorageSize  int64  `json:"baseBlockStorageSize"`
	DiskType              Code   `json:"diskType"`
	DiskDetailType        Code   `json:"diskDetailType"`
	GenerationCode        string `json:"generationCode"`

	PriceList []Price `json:"priceList"`

	softwareType map[string]interface{} `json:"softwareType"`
}

type PriceListAPIResponse struct {
	GetProductPriceListResponse struct {
		RequestID        string         `json:"request_id"`
		ReturnCode       int            `json:"return_code"`
		ReturnMessage    string         `returnMessage`
		TotalRows        int            `json:"totalRows"`
		ProductPriceList []ProductPrice `json:"productPriceList"`
	} `json:"getProductPriceListResponse"`

	Error *ErrorResponse `json:"error,omitempty"`
}

// =========================== For PriceList ============================

// =========================== Common ============================
type ErrorResponse struct {
	Code    string `json:"errorCode,omitempty"`
	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`
}

// =========================== Common ============================

const (
	BaseURL             string = "https://billingapi.apigw.ntruss.com/billing/v1"
	ProductListURL      string = "/product/getProductList"
	ProductPriceListURL string = "/product/getProductPriceList"
)

func (priceInfoHandler *NcpPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	cblogger.Info("NCP Classic Cloud driver: called ListProductFamily()!!")
	// API Guide : https://api.ncloud-docs.com/docs/platform-listprice-getproductlist

	if strings.EqualFold(regionName, "") {
		newErr := fmt.Errorf("Invalid regionName!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	bodyInUint8, err := priceInfoHandler.GetRequestBody(regionName, ProductListURL)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Body to Request : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var productListResp ProductListAPIResponse
	err = json.Unmarshal(bodyInUint8, &productListResp)
	if err != nil {
		newErr := fmt.Errorf("Failed to Unmarshal JSON : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if productListResp.Error != nil {
		newErr := fmt.Errorf("API Error Code: [%s], Message: [%s]", productListResp.Error.Code, productListResp.Error.Message)
		cblogger.Error(newErr.Error())
		return nil, nil
	}
	// log.Printf("### productListResp")
	// spew.Dump(productListResp)

	// # Remove Duplicated Product Code
	uniqueCodes := make(map[string]bool)
	uniqueProducts := []Product{}
	if len(productListResp.GetProductListResponse.ProductList) > 0 {
		for _, product := range productListResp.GetProductListResponse.ProductList {
			if _, exists := uniqueCodes[product.ItemKind.Code]; !exists {
				uniqueProducts = append(uniqueProducts, product)
				uniqueCodes[product.ItemKind.Code] = true
			}
		}
	}

	var productCodeList []string
	if len(uniqueProducts) > 0 {
		for _, uniqueProduct := range uniqueProducts {
			// fmt.Println("Code:", uniqueProduct.ItemKind.Code)
			productCodeList = append(productCodeList, uniqueProduct.ItemKind.Code)
		}
	}
	return productCodeList, nil
}

func (priceInfoHandler *NcpPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetPriceInfo()!!")
	// API Guide : https://api.ncloud-docs.com/docs/platform-listprice-getproductlist

	if strings.EqualFold(regionName, "") {
		newErr := fmt.Errorf("Invalid regionName!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// Check whether the presented ProductFamily exists.
	productList, err := priceInfoHandler.ListProductFamily(regionName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get ProductFamily : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	found := false
	for _, product := range productList {
		if strings.EqualFold(product, productFamily) {
			found = true
			break
		}
	}
	if found {
		cblogger.Infof("The ProductFamily '%s' is Included in the ProductFamily.\n", productFamily)
	} else {
		newErr := fmt.Errorf("The ProductFamily '%s' is Not Included in the ProductFamily.\n", productFamily)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	bodyInUint8, err := priceInfoHandler.GetRequestBodyWithProductCode(regionName, ProductPriceListURL, productFamily, filterList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Body to Request : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var priceListResp PriceListAPIResponse
	err = json.Unmarshal(bodyInUint8, &priceListResp)
	if err != nil {
		newErr := fmt.Errorf("Failed to unmarshal JSON : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	// log.Printf("### response")
	// spew.Dump(response)

	var priceList []irs.Price
	switch productFamily {
	case "SVR": // Server(VM)
		for _, product := range priceListResp.GetProductPriceListResponse.ProductPriceList {
			// fmt.Printf("ProductCode: %s\n", product.ProductCode)
			// fmt.Printf("ProductItemKindDetail Code: %s\n", product.ProductItemKindDetail.Code)
			// fmt.Printf("ProductItemKindDetail CodeName: %s\n", product.ProductItemKindDetail.CodeName)
			// fmt.Printf("ProductType CodeName: %s\n", product.ProductType.CodeName)
			// fmt.Printf("CpuCount: %d\n", product.CpuCount)
			// fmt.Printf("MemorySize: %d\n", product.MemorySize)
			// fmt.Printf("BaseBlockStorageSize: %d\n", product.BaseBlockStorageSize)
			// fmt.Printf("GpuCount: %d\n", product.GpuCount)
			// fmt.Printf("DiskType Code: %s\n", product.DiskType.Code)
			// fmt.Printf("DiskType CodeName: %s\n", product.DiskType.CodeName)
			// fmt.Println("------------------------------")

			var onDemand irs.OnDemand
			for _, price := range product.PriceList {
				if strings.EqualFold(price.Region.RegionCode, regionName) {
					priceString := fmt.Sprintf("%f", price.PriceValue)
					onDemand = irs.OnDemand{
						PricingId:   price.PriceNo,
						Unit:        price.Unit.CodeName,
						Currency:    price.PayCurrency.Code,
						Price:       priceString,
						Description: price.PriceDescription,
					}
					break
				}
			}

			// Skip if there's no OnDemand price
			if onDemand.PricingId == "" {
				continue
			}

			vCPUs := strconv.Itoa(product.CpuCount)
			vMemGb := strconv.FormatInt(product.MemorySize/(1024*1024*1024), 10)
			storageGB := strconv.FormatInt(product.BaseBlockStorageSize/(1024*1024*1024), 10)

			var gpuInfoList []irs.GpuInfo
			if product.GpuCount > 0 {
				aGPU := irs.GpuInfo{
					Count:          strconv.Itoa(product.GpuCount),
					MemSizeGB:      "-1",
					TotalMemSizeGB: "-1",
					Mfr:            "NA",
					Model:          "NA",
				}
				gpuInfoList = append(gpuInfoList, aGPU)
			}

			// Create KeyValueList for disk type information
			diskTypeInfo := []irs.KeyValue{
				{Key: "DiskType", Value: product.DiskType.CodeName},
				{Key: "DiskDetailType", Value: product.DiskDetailType.CodeName},
			}

			priceList = append(priceList, irs.Price{
				ZoneName: "NA",
				ProductInfo: irs.ProductInfo{
					ProductId: product.ProductCode,
					VMSpecInfo: irs.VMSpecInfo{
						Name:         product.ProductType.CodeName,
						VCpu:         irs.VCpuInfo{Count: vCPUs, ClockGHz: "-1"},
						MemSizeMiB:   vMemGb,
						DiskSizeGB:   storageGB,
						Gpu:          gpuInfoList,
						KeyValueList: diskTypeInfo,
					},
					Description:    product.ProductDescription,
					CSPProductInfo: product,
				},
				PriceInfo: irs.PriceInfo{
					OnDemand:     onDemand,
					CSPPriceInfo: product.PriceList,
				},
			})
		}

	case "BST": // Block Storage
		for _, product := range priceListResp.GetProductPriceListResponse.ProductPriceList {
			// fmt.Printf("ProductCode: %s\n", product.ProductCode)
			// fmt.Printf("ProductType Code: %s\n", product.ProductType.Code)
			// fmt.Printf("DiskDetailType CodeName: %s\n", product.DiskDetailType.CodeName)
			// fmt.Println("------------------------------")

			var onDemand irs.OnDemand
			for _, price := range product.PriceList {
				if strings.EqualFold(price.Region.RegionCode, regionName) {
					priceString := fmt.Sprintf("%f", price.PriceValue)
					onDemand = irs.OnDemand{
						PricingId:   price.PriceNo,
						Unit:        price.Unit.CodeName,
						Currency:    price.PayCurrency.Code,
						Price:       priceString,
						Description: price.PriceDescription,
					}
					break
				}
			}

			// Skip if there's no OnDemand price
			if onDemand.PricingId == "" {
				continue
			}

			priceList = append(priceList, irs.Price{
				ZoneName: "NA",
				ProductInfo: irs.ProductInfo{
					ProductId:      product.ProductCode,
					Description:    product.ProductDescription,
					CSPProductInfo: product,
				},
				PriceInfo: irs.PriceInfo{
					OnDemand:     onDemand,
					CSPPriceInfo: product.PriceList,
				},
			})
		}

	default:
		newErr := fmt.Errorf(productFamily + " is Not Supported Product Family on this driver yet!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	cloudPrice := irs.CloudPrice{
		Meta:       irs.Meta{Version: "0.1", Description: "NCP Classic Price Info"},
		CloudName:  "NCP",
		RegionName: regionName,
		PriceList:  priceList,
	}

	jsonData, err := json.MarshalIndent(cloudPrice, "", "    ")
	if err != nil {
		newErr := errors.New(fmt.Sprintf("Failed to Get PriceInfo Data : [%s]", err))
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return string(jsonData), nil
}

// This is necessary because NCP GoSDK does not support these PriceInfo APIs.
func (priceInfoHandler *NcpPriceInfoHandler) GetRequestBody(regionCode string, callURL string) ([]uint8, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetRequestBody()!!")

	// ### Ref for Auth.) https://api.ncloud-docs.com/docs/common-ncpapi
	// Set Query Parameters
	params := url.Values{}
	params.Add("responseFormatType", "json") // Note!! : 'json' or 'xml'
	params.Add("regionCode", regionCode)

	// Add Query Parameters to BaseURL
	fullURL := BaseURL + callURL + "?" + params.Encode()

	accessKey := priceInfoHandler.CredentialInfo.ClientId
	secretKey := priceInfoHandler.CredentialInfo.ClientSecret

	// Current time -> Calculated in Milli-Second
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	// Ref) https://github.com/NaverCloudPlatform/ncloud-sdk-go-v2/blob/master/services/vserver/api_client.go  line 269 ~ 270
	signer := hmac.NewSigner(secretKey, crypto.SHA256)
	signature, _ := signer.Sign("GET", fullURL, accessKey, timestamp) // Caution!! : Different from the general signature format.

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Request : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	req.Header.Set("x-ncp-apigw-timestamp", timestamp)
	req.Header.Set("x-ncp-iam-access-key", accessKey)
	req.Header.Set("x-ncp-apigw-signature-v2", signature)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		newErr := fmt.Errorf("Failed to Send Request : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := fmt.Errorf("Failed to Read Response Body : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// log.Printf("### body")
	// spew.Dump(body)

	// ### Convert []byte format of date to []unit8 format
	var dataInUint8 []uint8 = body // Caution!!

	return dataInUint8, nil
}

// This is necessary because NCP GoSDK does not support these PriceInfo APIs.
func (priceInfoHandler *NcpPriceInfoHandler) GetRequestBodyWithProductCode(regionCode string, callURL string, productCode string, filterList []irs.KeyValue) ([]uint8, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetRequestBodyWithProductCode()!!")

	// ### Ref for Auth.) https://api.ncloud-docs.com/docs/common-ncpapi
	// Set Query Parameters
	// NCP API Call URL Ex) : GET {API_URL}/product/getProductPriceList?regionCode=KR&productItemKindCode=VSVR&productName=6248R
	params := url.Values{}
	params.Add("responseFormatType", "json") // Note!! : 'json' or 'xml'
	params.Add("regionCode", regionCode)
	params.Add("productItemKindCode", productCode) // Ex) SVR or VSVR, ...

	if len(filterList) == 0 {
		filterList = nil
	} else {
		for _, filter := range filterList {
			params.Add(filter.Key, filter.Value)
		}
	}

	// Add Query Parameters to BaseURL
	fullURL := BaseURL + callURL + "?" + params.Encode()

	accessKey := priceInfoHandler.CredentialInfo.ClientId
	secretKey := priceInfoHandler.CredentialInfo.ClientSecret

	// Current time -> Calculated in Milli-Second
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	// Ref) https://github.com/NaverCloudPlatform/ncloud-sdk-go-v2/blob/master/services/vserver/api_client.go  line 269 ~ 270
	signer := hmac.NewSigner(secretKey, crypto.SHA256)
	signature, _ := signer.Sign("GET", fullURL, accessKey, timestamp) // Caution!! : Different from the general signature format.

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Request : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	req.Header.Set("x-ncp-apigw-timestamp", timestamp)
	req.Header.Set("x-ncp-iam-access-key", accessKey)
	req.Header.Set("x-ncp-apigw-signature-v2", signature)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		newErr := fmt.Errorf("Failed to Send Request : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := fmt.Errorf("Failed to Read Response Body : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// log.Printf("### body")
	// spew.Dump(body)

	// ### Convert []byte format of date to []unit8 format
	var dataInUint8 []uint8 = body // Caution!!

	return dataInUint8, nil
}
