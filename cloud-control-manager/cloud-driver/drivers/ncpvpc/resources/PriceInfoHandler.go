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

	// "log"
	// "github.com/davecgh/go-spew/spew"

	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcPriceInfoHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
}

// Already declared in CommonNcpFunc.go
// var cblogger *logrus.Logger
func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC VMHandler")
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

// # Get Product 'Name' of all Products instead of Product 'Code'
func (priceInfoHandler *NcpVpcPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	cblogger.Info("NCP VPC Cloud driver: called ListProductFamily()!!")
	// API Guide : https://api.ncloud-docs.com/docs/platform-listprice-getproductlist

	if strings.EqualFold(regionName, "") {
		newErr := fmt.Errorf("Invalid regionName!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	productItemKindList, err := priceInfoHandler.getProductItemKindList(regionName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get ProductItemKind List : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// log.Printf("### productItemKindList")
	// spew.Dump(productItemKindList)	

	var productCodeNameList []string
	if len(productItemKindList) > 0 {
		for _, productItemKind := range productItemKindList {
			productCodeNameList = append(productCodeNameList, productItemKind.CodeName)
		}
	}

	return productCodeNameList, nil
}

func (priceInfoHandler *NcpVpcPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetPriceInfo()!!")
	// API Guide : https://api.ncloud-docs.com/docs/platform-listprice-getproductlist

	if strings.EqualFold(productFamily, "") {
		newErr := fmt.Errorf("Invalid productFamily Name!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if strings.EqualFold(regionName, "") {
		newErr := fmt.Errorf("Invalid regionName!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// Check whether the presented ProductFamily exists.
	productItemKindList, err := priceInfoHandler.getProductItemKindList(regionName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get ProductItemKind List : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	
	found := false
	for _, productItemKind := range productItemKindList {
		if strings.EqualFold(productItemKind.CodeName, productFamily) {
			found = true
			break
		}
	}
	if found {
		fmt.Printf("The ProductFamily '%s' is Included in the ProductFamily.\n", productFamily)
	} else {
		newErr := fmt.Errorf("The ProductFamily '%s' is Not Included in the ProductFamily.\n", productFamily)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	productCode, err := priceInfoHandler.getProductCodeWithProductName(productFamily, regionName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Product Code with the Product Family Name : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	productPriceList, err := priceInfoHandler.getProductPriceListWithProductCode(regionName, ProductPriceListURL, productCode, filterList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get ProductPrice List : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	// log.Printf("### productPriceList")
	// spew.Dump(productPriceList)

	var priceList []irs.Price
	switch productCode { // Not productFamily
	case "SVR": // Server(VM)
		for _, productPrice := range productPriceList {

			var regionCode string
			for _, price := range productPrice.PriceList {
				if strings.EqualFold(price.Region.RegionCode, regionName) {
					regionCode = price.Region.RegionCode
					break
				}
				fmt.Printf("RegionCode: %s\n", price.Region.RegionCode)
			}

			var pricingPolicies []irs.PricingPolicies
			for _, price := range productPrice.PriceList {
				priceString := fmt.Sprintf("%f", price.PriceValue)
				pricingPolicies = append(pricingPolicies, irs.PricingPolicies{
					PricingId:         price.PriceNo,
					PricingPolicy:     price.PriceType.CodeName,
					Unit:              price.Unit.CodeName,
					Currency:          price.PayCurrency.Code,
					Price:             priceString,
					Description:       price.PriceDescription,
					PricingPolicyInfo: nil,
				})
			}

			vCPUs := strconv.Itoa(productPrice.CpuCount)
			vMemGb := strconv.FormatInt(productPrice.MemorySize/(1024*1024*1024), 10)
			storageGB := strconv.FormatInt(productPrice.BaseBlockStorageSize/(1024*1024*1024), 10)
			vGPUs := strconv.Itoa(productPrice.GpuCount)

			priceList = append(priceList, irs.Price{
				ProductInfo: irs.ProductInfo{
					ProductId:       productPrice.ProductCode,
					RegionName:      regionCode,
					InstanceType:    productPrice.ProductType.CodeName,
					Vcpu:            vCPUs,
					Memory:          vMemGb,
					Storage:         storageGB,
					Gpu:             vGPUs,
					GpuMemory:       "N/A",
					OperatingSystem: "N/A",
					PreInstalledSw:  "N/A",
					VolumeType:      productPrice.DiskType.CodeName,
					StorageMedia:    productPrice.DiskDetailType.CodeName,
					Description:     productPrice.ProductName, // Some items do not give 'ProductDescription' info
					CSPProductInfo:  productPrice,
				},
				PriceInfo: irs.PriceInfo{
					PricingPolicies: pricingPolicies,
					CSPPriceInfo:    productPrice.PriceList,
				},
			})

		}

	case "BST": // Block storage
		for _, productPrice := range productPriceList {

			var regionCode string
			for _, price := range productPrice.PriceList {
				if strings.EqualFold(price.Region.RegionCode, regionName) {
					regionCode = price.Region.RegionCode
					break
				}
				fmt.Printf("RegionCode: %s\n", price.Region.RegionCode)
			}

			var pricingPolicies []irs.PricingPolicies
			for _, price := range productPrice.PriceList {
				priceString := fmt.Sprintf("%f", price.PriceValue)
				pricingPolicies = append(pricingPolicies, irs.PricingPolicies{
					PricingId:         price.PriceNo,
					PricingPolicy:     price.PriceType.CodeName,
					Unit:              price.Unit.CodeName,
					Currency:          price.PayCurrency.Code,
					Price:             priceString,
					Description:       price.PriceDescription,
					PricingPolicyInfo: nil,
				})
			}

			priceList = append(priceList, irs.Price{
				ProductInfo: irs.ProductInfo{
					ProductId:           productPrice.ProductCode,
					RegionName:          regionCode,
					VolumeType:          productPrice.ProductType.CodeName,
					StorageMedia:        productPrice.DiskDetailType.CodeName,
					MaxVolumeSize:       "N/A",
					MaxIOPSVolume:       "N/A",
					MaxThroughputVolume: "N/A",
					Description:         productPrice.ProductDescription,
					CSPProductInfo:      productPrice,
				},
				PriceInfo: irs.PriceInfo{
					PricingPolicies: pricingPolicies,
					CSPPriceInfo:    productPrice.PriceList,
				},
			})

		}

	default:
		newErr := fmt.Errorf(productFamily + " is Not Supported Product Family on this driver yet!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	cloudPriceData := irs.CloudPriceData{
		Meta: irs.Meta{
			Version:     "v0.1",
			Description: "Multi-Cloud Price Info",
		},
		CloudPriceList: []irs.CloudPrice{
			{
				CloudName: "NCP VPC",
				PriceList: priceList,
			},
		},
	}

	jsonData, err := json.MarshalIndent(cloudPriceData, "", "    ")
	// jsonData, err := json.Marshal(cloudPriceData)
	if err != nil {
		newErr := errors.New(fmt.Sprintf("Failed to Get PriceInfo Data : [%s]", err))
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return string(jsonData), nil
}

// This is necessary because NCP GoSDK does not support these PriceInfo APIs.
func (priceInfoHandler *NcpVpcPriceInfoHandler) getProductCodeList(regionCode string, callURL string) ([]string, error) {
	cblogger.Info("NCP VPC Cloud driver: called getProductCodeList()!!")

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
	var bodyInUint8 []uint8 = body // Caution!!

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
	} else {
		return nil, nil	
	}
	return productCodeList, nil
}

func (priceInfoHandler *NcpVpcPriceInfoHandler) getProductItemKindList(regionName string) ([]ProductItemKind, error) {
	cblogger.Info("NCP VPC Cloud driver: called getProductItemKindList()!!")
	// API Guide : https://api.ncloud-docs.com/docs/platform-listprice-getproductlist

	if strings.EqualFold(regionName, "") {
		newErr := fmt.Errorf("Invalid regionName!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	productCodeList, err := priceInfoHandler.getProductCodeList(regionName, ProductListURL)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get ProductCode List : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// log.Printf("### productCodeList")
	// spew.Dump(productCodeList)		

	uniqueCodeNames := make(map[string]bool)
	var productItemKindList []ProductItemKind

	// # Since some productItemKind.codeName is empty in getProductList (API) supporting from NCP
	if len(productCodeList) > 0 {
		for _, productCode := range productCodeList {
			productPriceList, err := priceInfoHandler.getProductPriceListWithProductCode(regionName, ProductPriceListURL, productCode, nil)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get ProductPrice List : [%v]", err)
				cblogger.Error(newErr.Error())
				return nil, newErr
			}
			// log.Printf("### productPriceList")
			// spew.Dump(productPriceList)		

			// # Remove Duplicated CodeName
			for _, productPrice := range productPriceList {
				if _, exists := uniqueCodeNames[productPrice.ProductItemKind.CodeName]; !exists {
					newProductItemKind := ProductItemKind { 
						Code: productPrice.ProductItemKind.Code,
						CodeName: productPrice.ProductItemKind.CodeName,
					}
					productItemKindList = append(productItemKindList, newProductItemKind)
					uniqueCodeNames[productPrice.ProductItemKind.CodeName] = true
				}
			}			
		}
	}
	// log.Printf("### productItemKindList")
	// spew.Dump(productItemKindList)		

	return productItemKindList, nil
}


// This is necessary because NCP GoSDK does not support these PriceInfo APIs.
func (priceInfoHandler *NcpVpcPriceInfoHandler) getProductPriceListWithProductCode(regionCode string, callURL string, productCode string, filterList []irs.KeyValue) ([]ProductPrice, error) {
	cblogger.Info("NCP VPC Cloud driver: called getProductPriceListWithProductCode()!!")

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

	// ### Convert []byte format of date to []unit8 format
	var bodyInUint8 []uint8 = body // Caution!!

	var priceListResp PriceListAPIResponse
	err = json.Unmarshal(bodyInUint8, &priceListResp)
	if err != nil {
		newErr := fmt.Errorf("Failed to unmarshal JSON : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	return priceListResp.GetProductPriceListResponse.ProductPriceList, nil
}


func (priceInfoHandler *NcpVpcPriceInfoHandler) getProductCodeWithProductName(productName string, regionName string) (string, error) {
	cblogger.Info("NCP VPC Cloud driver: called getProductCodeWithProductName()!!")
	// API Guide : https://api.ncloud-docs.com/docs/platform-listprice-getproductlist

	if strings.EqualFold(productName, "") {
		newErr := fmt.Errorf("Invalid productName!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if strings.EqualFold(regionName, "") {
		newErr := fmt.Errorf("Invalid regionName!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	productItemKindList, err := priceInfoHandler.getProductItemKindList(regionName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get ProductItemKind List : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var productCode string
	if len(productItemKindList) > 0 {
		for _, productItemKind := range productItemKindList {
			if strings.EqualFold(productItemKind.CodeName, productName) {
				productCode = productItemKind.Code
				break				
			}		
		}
	}
	return productCode, nil
}
