package resources

import (
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/hmac"

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

func init() {
	cblogger = cblog.GetLogger("NCP VPC VMHandler")
}

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

type Code struct {
	Code     string `json:"code"`
	CodeName string `json:"codeName"`
}

type Region struct {
	RegionNo   int    `json:"regionNo"`
	RegionCode string `json:"regionCode"`
	RegionName string `json:"regionName"`
}

type Price struct {
	PriceNo                string  `json:"priceNo"`
	PriceType              Code    `json:"priceType"`
	Region                 Region  `json:"region"`
	ChargingUnitType       Code    `json:"chargingUnitType"`
	RatingUnitType         Code    `json:"ratingUnitType"`
	ChargingUnitBasicValue string  `json:"chargingUnitBasicValue"`
	Unit                   Code    `json:"unit"`
	PriceValue             float32 `json:"price"`
	ConditionType          Code    `json:"conditionType"`
	ConditionPrice         float32 `json:"conditionPrice"`
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

type ErrorResponse struct {
	Code    string `json:"errorCode,omitempty"`
	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`
}

const (
	BaseURL             string = "https://billingapi.apigw.ntruss.com/billing/v1"
	ProductListURL      string = "/product/getProductList"
	ProductPriceListURL string = "/product/getProductPriceList"
)

func (priceInfoHandler *NcpVpcPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	cblogger.Info("NCP VPC Cloud driver: called ListProductFamily()!!")

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

	var productCodeNameList []string
	if len(productItemKindList) > 0 {
		for _, productItemKind := range productItemKindList {
			productCodeNameList = append(productCodeNameList, productItemKind.CodeName)
		}
	}

	return productCodeNameList, nil
}

func extractGPUMemoryFromProductName(productName string) (int64, error) {
	re := regexp.MustCompile(`GPUMemory\s+(\d+)GB`)
	matches := re.FindStringSubmatch(productName)

	if len(matches) > 1 {
		memSizeStr := matches[1]
		memSize, err := strconv.ParseInt(memSizeStr, 10, 64)
		if err != nil {
			return -1, err
		}
		return memSize, nil
	}

	return -1, fmt.Errorf("GPUMemory information not found in product name")
}

func (priceInfoHandler *NcpVpcPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetPriceInfo()!!")

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

	productFamily = decodeURLString(productFamily)

	productItemKindList, err := priceInfoHandler.getProductItemKindList(regionName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get ProductItemKind List : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	found := false
	var productCode string
	for _, productItemKind := range productItemKindList {
		if strings.EqualFold(productItemKind.CodeName, productFamily) {
			found = true
			productCode = productItemKind.Code
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

	productPriceList, err := priceInfoHandler.getProductPriceListWithProductCode(regionName, ProductPriceListURL, productCode, filterList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get ProductPrice List : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var priceList []irs.Price
	switch productCode {
	case "VSVR":
		vmSpecMap, err := priceInfoHandler.getVMSpecMap(regionName)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get VMSpec information: [%v]", err)
			cblogger.Error(newErr.Error())
			return "", newErr
		}

		for _, productPrice := range productPriceList {

			var onDemand irs.OnDemand
			for _, price := range productPrice.PriceList {
				if strings.EqualFold(price.PriceType.Code, "MTRAT") {
					priceString := fmt.Sprintf("%.4f", price.PriceValue)

					unitName := price.Unit.CodeName
					if strings.EqualFold(unitName, "Usage time (per hour)") {
						unitName = "Hour"
					}

					onDemand = irs.OnDemand{
						PricingId:   price.PriceNo,
						Unit:        unitName,
						Currency:    price.PayCurrency.Code,
						Price:       priceString,
						Description: price.PriceDescription,
					}
					break
				}
			}

			if onDemand.Price == "" {
				continue
			}

			vCPUs := strconv.Itoa(productPrice.CpuCount)
			vMemGb := strconv.FormatInt(productPrice.MemorySize/(1024*1024), 10)
			storageGB := strconv.FormatInt(productPrice.BaseBlockStorageSize/(1024*1024*1024), 10)

			var gpuInfoList []irs.GpuInfo
			if productPrice.GpuCount > 0 {
				totalGpuMemGB := "-1"
				singleGpuMemGB := "-1"

				gpuMemSize, err := extractGPUMemoryFromProductName(productPrice.ProductName)
				if err == nil && gpuMemSize > 0 {
					totalGpuMemGB = strconv.FormatInt(gpuMemSize, 10)

					if productPrice.GpuCount > 0 {
						singleGpuMemPerGPU := gpuMemSize / int64(productPrice.GpuCount)
						singleGpuMemGB = strconv.FormatInt(singleGpuMemPerGPU, 10)
					}
				}

				aGPU := irs.GpuInfo{
					Count:          strconv.Itoa(productPrice.GpuCount),
					MemSizeGB:      singleGpuMemGB,
					TotalMemSizeGB: totalGpuMemGB,
					Mfr:            "NA",
					Model:          "NA",
				}
				gpuInfoList = append(gpuInfoList, aGPU)
			}

			specName := productPrice.ProductType.CodeName
			if serverSpecCode, exists := vmSpecMap[productPrice.ProductCode]; exists {
				specName = serverSpecCode
			} else {
				cblogger.Infof("Could not find matching ServerSpecCode for ProductCode: %s, using ProductType.CodeName as fallback", productPrice.ProductCode)
			}

			priceList = append(priceList, irs.Price{
				ZoneName: "NA",
				ProductInfo: irs.ProductInfo{
					ProductId:      productPrice.ProductCode,
					Description:    productPrice.ProductName,
					CSPProductInfo: productPrice,
					VMSpecInfo: irs.VMSpecInfo{
						Name:       specName,
						VCpu:       irs.VCpuInfo{Count: vCPUs, ClockGHz: "-1"},
						MemSizeMiB: vMemGb,
						DiskSizeGB: storageGB,
						Gpu:        gpuInfoList,
					},
				},
				PriceInfo: irs.PriceInfo{
					OnDemand:     onDemand,
					CSPPriceInfo: productPrice.PriceList,
				},
			})
		}

	case "BST":
		for _, productPrice := range productPriceList {

			var onDemand irs.OnDemand
			for _, price := range productPrice.PriceList {
				if strings.EqualFold(price.PriceType.Code, "MTRAT") {
					priceString := fmt.Sprintf("%.4f", price.PriceValue)

					unitName := price.Unit.CodeName
					if strings.EqualFold(unitName, "Usage time (per hour)") {
						unitName = "Hour"
					}

					onDemand = irs.OnDemand{
						PricingId:   price.PriceNo,
						Unit:        unitName,
						Currency:    price.PayCurrency.Code,
						Price:       priceString,
						Description: price.PriceDescription,
					}
					break
				}
			}

			if onDemand.Price == "" {
				continue
			}

			priceList = append(priceList, irs.Price{
				ZoneName: "NA",
				ProductInfo: irs.ProductInfo{
					ProductId:      productPrice.ProductCode,
					Description:    productPrice.ProductDescription,
					CSPProductInfo: productPrice,
				},
				PriceInfo: irs.PriceInfo{
					OnDemand:     onDemand,
					CSPPriceInfo: productPrice.PriceList,
				},
			})
		}

	default:
		newErr := fmt.Errorf(productFamily + " is Not Supported Product Family on this driver yet!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	cloudPrice := irs.CloudPrice{
		Meta:       irs.Meta{Version: "0.5", Description: "Multi-Cloud Price Info"},
		CloudName:  "NCPVPC",
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

func (priceInfoHandler *NcpVpcPriceInfoHandler) getVMSpecMap(regionName string) (map[string]string, error) {
	cblogger.Info("NCP VPC Cloud driver: called getVMSpecMap()!!")

	vmSpecHandler := NcpVpcVMSpecHandler{
		CredentialInfo: priceInfoHandler.CredentialInfo,
		RegionInfo: idrv.RegionInfo{
			Region: regionName,
			Zone:   "",
		},
		VMClient: priceInfoHandler.VMClient,
	}

	vmSpecList, err := vmSpecHandler.ListVMSpec()
	if err != nil {
		newErr := fmt.Errorf("Failed to get VM Spec list: [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	vmSpecMap := make(map[string]string)

	for _, spec := range vmSpecList {
		for _, kv := range spec.KeyValueList {
			if kv.Key == "ServerProductCode" {
				vmSpecMap[kv.Value] = spec.Name
				break
			}
		}
	}

	if len(vmSpecMap) == 0 {
		orgVMSpecList, err := vmSpecHandler.getNcpVpcVMSpecList()
		if err != nil {
			newErr := fmt.Errorf("Failed to get original VM Spec list: [%v]", err)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}

		for _, spec := range orgVMSpecList {
			if spec.ServerProductCode != nil && spec.ServerSpecCode != nil {
				vmSpecMap[*spec.ServerProductCode] = *spec.ServerSpecCode
			}
		}
	}

	return vmSpecMap, nil
}

func decodeURLString(encodedString string) string {
	decoded, err := url.QueryUnescape(encodedString)
	if err == nil {
		return decoded
	}

	replacements := map[string]string{
		"%20": " ",  // space
		"%21": "!",  // exclamation mark
		"%22": "\"", // double quote
		"%27": "'",  // single quote
		"%28": "(",  // opening parenthesis
		"%29": ")",  // closing parenthesis
		"%2C": ",",  // comma
		"%3A": ":",  // colon
		"%3B": ";",  // semicolon
		"%3C": "<",  // less than
		"%3E": ">",  // greater than
		"%3D": "=",  // equals
		"%3F": "?",  // question mark
		"%40": "@",  // at sign
		"%5B": "[",  // opening bracket
		"%5D": "]",  // closing bracket
		"%7B": "{",  // opening brace
		"%7D": "}",  // closing brace
		"%25": "%",  // percent sign
	}

	result := encodedString
	for encoded, decoded := range replacements {
		result = strings.ReplaceAll(result, encoded, decoded)
	}

	return result
}

func (priceInfoHandler *NcpVpcPriceInfoHandler) getProductCodeList(regionCode string, callURL string) ([]string, error) {
	cblogger.Info("NCP VPC Cloud driver: called getProductCodeList()!!")

	params := url.Values{}
	params.Add("responseFormatType", "json")
	params.Add("regionCode", regionCode)

	fullURL := BaseURL + callURL + "?" + params.Encode()

	accessKey := priceInfoHandler.CredentialInfo.ClientId
	secretKey := priceInfoHandler.CredentialInfo.ClientSecret

	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	signer := hmac.NewSigner(secretKey, crypto.SHA256)
	signature, _ := signer.Sign("GET", fullURL, accessKey, timestamp)

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

	var bodyInUint8 []uint8 = body

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
			productCodeList = append(productCodeList, uniqueProduct.ItemKind.Code)
		}
	} else {
		return nil, nil
	}
	return productCodeList, nil
}

func (priceInfoHandler *NcpVpcPriceInfoHandler) getProductItemKindList(regionName string) ([]ProductItemKind, error) {
	cblogger.Info("NCP VPC Cloud driver: called getProductItemKindList()!!")

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

	uniqueCodeNames := make(map[string]bool)
	var productItemKindList []ProductItemKind

	if len(productCodeList) > 0 {
		for _, productCode := range productCodeList {
			productPriceList, err := priceInfoHandler.getProductPriceListWithProductCode(regionName, ProductPriceListURL, productCode, nil)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get ProductPrice List : [%v]", err)
				cblogger.Error(newErr.Error())
				return nil, newErr
			}

			for _, productPrice := range productPriceList {
				if _, exists := uniqueCodeNames[productPrice.ProductItemKind.CodeName]; !exists {
					newProductItemKind := ProductItemKind{
						Code:     productPrice.ProductItemKind.Code,
						CodeName: productPrice.ProductItemKind.CodeName,
					}
					productItemKindList = append(productItemKindList, newProductItemKind)
					uniqueCodeNames[productPrice.ProductItemKind.CodeName] = true
				}
			}
		}
	}

	return productItemKindList, nil
}

func (priceInfoHandler *NcpVpcPriceInfoHandler) getProductPriceListWithProductCode(regionCode string, callURL string, productCode string, filterList []irs.KeyValue) ([]ProductPrice, error) {
	cblogger.Info("NCP VPC Cloud driver: called getProductPriceListWithProductCode()!!")

	params := url.Values{}
	params.Add("responseFormatType", "json")
	params.Add("regionCode", regionCode)
	params.Add("productItemKindCode", productCode)
	params.Add("payCurrencyCode", "USD")

	if len(filterList) == 0 {
		filterList = nil
	} else {
		for _, filter := range filterList {
			params.Add(filter.Key, filter.Value)
		}
	}

	fullURL := BaseURL + callURL + "?" + params.Encode()

	accessKey := priceInfoHandler.CredentialInfo.ClientId
	secretKey := priceInfoHandler.CredentialInfo.ClientSecret

	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	signer := hmac.NewSigner(secretKey, crypto.SHA256)
	signature, _ := signer.Sign("GET", fullURL, accessKey, timestamp)

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

	var bodyInUint8 []uint8 = body

	var priceListResp PriceListAPIResponse
	err = json.Unmarshal(bodyInUint8, &priceListResp)
	if err != nil {
		newErr := fmt.Errorf("Failed to unmarshal JSON : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	return priceListResp.GetProductPriceListResponse.ProductPriceList, nil
}
