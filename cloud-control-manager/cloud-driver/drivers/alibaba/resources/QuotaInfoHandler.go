// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba Driver.
//
// by CB-Spider Team, 2025.07.

package resources

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/quotas"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaQuotaInfoHandler struct {
	Region      idrv.RegionInfo
	QuotaClient *quotas.Client
}

// ListServiceType returns the list of Alibaba product codes that have quota
// information available via the Quota Center API (ListProducts).
func (handler *AlibabaQuotaInfoHandler) ListServiceType() ([]string, error) {
	cblogger.Info("Alibaba Driver: called ListServiceType()")

	var serviceTypes []string
	nextToken := ""
	for {
		request := quotas.CreateListProductsRequest()
		request.MaxResults = requests.NewInteger(100)
		if nextToken != "" {
			request.NextToken = nextToken
		}

		response, err := handler.QuotaClient.ListProducts(request)
		if err != nil {
			return nil, fmt.Errorf("Alibaba ListServiceType: %w", err)
		}

		for _, p := range response.ProductInfo {
			if p.ProductCode != "" && strings.EqualFold(strings.TrimSpace(p.CommonQuotaSupport), "support") {
				serviceTypes = append(serviceTypes, p.ProductCode)
			}
		}

		if response.NextToken == "" || len(response.ProductInfo) == 0 {
			break
		}
		nextToken = response.NextToken
	}

	sort.Strings(serviceTypes)
	return serviceTypes, nil
}

// GetQuotaInfo retrieves ALL quota items for the given Alibaba product code.
// No filtering or name-mapping is performed; CSP-original values are passed through.
func (handler *AlibabaQuotaInfoHandler) GetQuotaInfo(serviceType string) (irs.QuotaInfo, error) {
	cblogger.Infof("Alibaba Driver: called GetQuotaInfo(serviceType=%s)", serviceType)

	quotaInfo := irs.QuotaInfo{
		CSP:    "ALIBABA",
		Region: handler.Region.Region,
	}

	quotaItems, err := handler.listProductQuotasAll(serviceType)
	if err != nil {
		return quotaInfo, fmt.Errorf("Alibaba GetQuotaInfo(%s): %w", serviceType, err)
	}

	var quotas []irs.Quota

	for _, q := range quotaItems {
		// Build QuotaName: use CSP's QuotaName directly
		quotaName := strings.TrimSpace(q.QuotaName)
		if quotaName == "" {
			quotaName = q.QuotaActionCode
		}
		if quotaName == "" {
			continue
		}

		// Limit
		limit := strconv.FormatFloat(q.TotalQuota, 'f', -1, 64)

		// Used & Available
		used := "NA"
		available := "NA"
		if q.Consumable {
			used = strconv.FormatFloat(q.TotalUsage, 'f', -1, 64)
			avail := q.TotalQuota - q.TotalUsage
			if avail >= 0 {
				available = strconv.FormatFloat(avail, 'f', -1, 64)
			}
		}

		// Unit: use CSP-provided value; set "NA" if not provided
		unit := "NA"
		if strings.TrimSpace(q.QuotaUnit) != "" {
			unit = strings.TrimSpace(q.QuotaUnit)
		}

		// Description: combine product code and CSP quota description
		desc := fmt.Sprintf("%s", strings.TrimSpace(q.QuotaDescription))

		rq := irs.Quota{
			QuotaName:   quotaName,
			Limit:       limit,
			Used:        used,
			Available:   available,
			Unit:        unit,
			Description: desc,
		}
		quotas = append(quotas, rq)
	}

	quotaInfo.Quotas = quotas
	return quotaInfo, nil
}

// listProductQuotasAll retrieves all quota items for a given product code,
// handling pagination via NextToken.
// It first attempts with regionId dimension.
//   - If the product does not support the regionId dimension key
//     (QUOTA.DIMENSION.UNSUPPORT), it retries without dimensions.
//   - If the product supports regionId but not the current region value
//     (QUOTA.DIMENSION.VALUE.UNSUPPORT), it returns an explicit region
//     availability error.
func (handler *AlibabaQuotaInfoHandler) listProductQuotasAll(productCode string) ([]quotas.QuotasItemInListProductQuotas, error) {
	result, err := handler.listProductQuotasPaginated(productCode, true)
	if err != nil && strings.Contains(err.Error(), "QUOTA.DIMENSION.UNSUPPORT") {
		cblogger.Infof("Alibaba QuotaInfoHandler: product %s does not support the regionId dimension filter for region %s, retrying without dimensions", productCode, handler.Region.Region)
		return handler.listProductQuotasPaginated(productCode, false)
	}
	if err != nil && strings.Contains(err.Error(), "QUOTA.DIMENSION.VALUE.UNSUPPORT") {
		return nil, fmt.Errorf("service %s is not available in region %s: %w", productCode, handler.Region.Region, err)
	}
	return result, err
}

// listProductQuotasPaginated is the internal paginated fetcher.
// If useRegionDimension is true, it sets the regionId dimension filter.
func (handler *AlibabaQuotaInfoHandler) listProductQuotasPaginated(productCode string, useRegionDimension bool) ([]quotas.QuotasItemInListProductQuotas, error) {
	var allQuotas []quotas.QuotasItemInListProductQuotas

	nextToken := ""
	for {
		request := quotas.CreateListProductQuotasRequest()
		request.ProductCode = productCode
		request.MaxResults = requests.NewInteger(100)
		if nextToken != "" {
			request.NextToken = nextToken
		}

		if useRegionDimension {
			// Set Dimensions for region-scoped quotas
			request.Dimensions = &[]quotas.ListProductQuotasDimensions{
				{
					Key:   "regionId",
					Value: handler.Region.Region,
				},
			}
		}

		response, err := handler.QuotaClient.ListProductQuotas(request)
		if err != nil {
			return allQuotas, fmt.Errorf("ListProductQuotas(%s): %w", productCode, err)
		}

		allQuotas = append(allQuotas, response.Quotas...)

		if response.NextToken == "" || len(response.Quotas) == 0 {
			break
		}
		nextToken = response.NextToken
	}

	return allQuotas, nil
}
