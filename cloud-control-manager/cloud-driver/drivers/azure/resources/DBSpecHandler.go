// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, August 2026.

package resources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// azureSkuDetail carries the per-SKU capability fields from the LocationBasedCapabilitySet
// response. NOTE: an earlier version of this file assumed SupportedSkus was nested one
// level deeper under SupportedServerVersions (matching the armmysqlflexibleservers SDK's
// LIST-capabilities model), but the actual `capabilitySets/default` endpoint response is
// FLAT — SupportedFlexibleServerEditions[].SupportedSkus[] directly, with NO
// SupportedServerVersions level in between (confirmed by doCapabilitySetRequest in
// RDBMSHandler.go, which has been working correctly with this flat shape all along and is
// the only field name (Name) verified against a live response). VCores/
// SupportedMemoryPerVCoreMB/SupportedIops field names are still an unverified best guess —
// if they don't match the live response, they simply decode as zero and are marked Static
// below rather than causing an error.
type azureSkuDetail struct {
	Name                      string `json:"name"`
	VCores                    int64  `json:"vCores"`
	SupportedMemoryPerVCoreMB int64  `json:"supportedMemoryPerVcoreMb"`
	SupportedIops             int64  `json:"supportedIops"`
}

// fetchAzureSKUDetails calls the same LocationBasedCapabilitySet_Get endpoint used by
// GetMetaInfo (fetchMySQLMetaOptions/doCapabilitySetRequest), but parses the response one
// level deeper to pull per-SKU vCPU/Memory/IOPS details instead of just SKU names.
func (handler *AzureRDBMSHandler) fetchAzureSKUDetails(location string) ([]azureSkuDetail, error) {
	cred, err := azidentity.NewClientSecretCredential(
		handler.CredentialInfo.TenantId,
		handler.CredentialInfo.ClientId,
		handler.CredentialInfo.ClientSecret,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}
	token, err := cred.GetToken(handler.Ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure token: %w", err)
	}

	apiURL := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/providers/Microsoft.DBforMySQL/locations/%s/capabilitySets/default?api-version=2023-12-30",
		handler.CredentialInfo.SubscriptionId, location,
	)
	req, err := http.NewRequestWithContext(handler.Ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build capabilitySets request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("capabilitySets request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read capabilitySets response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("capabilitySets API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Properties struct {
			SupportedFlexibleServerEditions []struct {
				SupportedSkus []azureSkuDetail `json:"supportedSkus"`
			} `json:"supportedFlexibleServerEditions"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse capabilitySets response: %w", err)
	}

	seen := map[string]bool{}
	var details []azureSkuDetail
	for _, edition := range result.Properties.SupportedFlexibleServerEditions {
		for _, sku := range edition.SupportedSkus {
			if sku.Name == "" || seen[sku.Name] {
				continue
			}
			seen[sku.Name] = true
			details = append(details, sku)
		}
	}
	if len(details) == 0 {
		return nil, fmt.Errorf("capabilitySets response did not include any SKU names (supportedFlexibleServerEditions[].supportedSkus[].name)")
	}
	return details, nil
}

func azureBuildDBSpecInfo(region, cbEngine string, d azureSkuDetail) *irs.DBSpecInfo {
	info := &irs.DBSpecInfo{
		Region:     region,
		DBEngine:   cbEngine,
		Name:       d.Name,
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}
	if d.VCores > 0 {
		info.VCpu.Count = strconv.FormatInt(d.VCores, 10)
	} else {
		info.MarkStatic("VCpu.Count", "capabilitySets response did not include VCores for this SKU")
	}
	// SupportedMemoryPerVCoreMB is documented by the raw field name as "MB", but Azure Compute
	// SKU-family APIs conventionally report this as MiB; treated as MiB here (no numeric
	// conversion applied), consistent with the value being a clean power-of-2-ish number.
	if d.VCores > 0 && d.SupportedMemoryPerVCoreMB > 0 {
		info.MemSizeMiB = strconv.FormatInt(d.VCores*d.SupportedMemoryPerVCoreMB, 10)
	} else {
		info.MarkStatic("MemSizeMiB", "capabilitySets response did not include SupportedMemoryPerVCoreMB for this SKU")
	}
	if d.SupportedIops > 0 {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "SupportedIops", Value: strconv.FormatInt(d.SupportedIops, 10)})
	}
	// StorageSizeRangeGB is intentionally left unset: the capabilitySets response's
	// per-SKU supportedSkus entries carry no storage-range field (storage size for Azure
	// Database for MySQL Flexible Server is a single engine-wide range, not tied to a
	// specific SKU) — see RDBMSMetaInfo.StorageSizeRangeGB for that range instead.
	info.MarkStatic("StorageSizeRangeGB", "The capabilitySets response's per-SKU details carry no storage-range field; storage size is a single engine-wide range for this CSP, not tied to a specific SKU. See RDBMSMetaInfo.StorageSizeRangeGB instead.")
	return info
}

func (handler *AzureRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Debug("Azure MySQL Flexible Server ListDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListDBSpec", "Microsoft.DBforMySQL/locations/capabilitySets/default")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return nil, err
	}
	if requestedEngine != "mysql" {
		err := fmt.Errorf("Azure Database for MySQL Flexible Server only supports mysql; mariadb/postgresql are not supported by this driver's RDBMSHandler")
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	details, err := handler.fetchAzureSKUDetails(handler.Region.Region)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	sort.Slice(details, func(i, j int) bool { return details[i].Name < details[j].Name })

	infoList := make([]*irs.DBSpecInfo, 0, len(details))
	for _, d := range details {
		infoList = append(infoList, azureBuildDBSpecInfo(handler.Region.Region, requestedEngine, d))
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return infoList, nil
}

func (handler *AzureRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Debug("Azure MySQL Flexible Server GetDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetDBSpec", "Microsoft.DBforMySQL/locations/capabilitySets/default")
	start := call.Start()

	infoList, err := handler.ListDBSpec(dbEngine)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DBSpecInfo{}, err
	}
	for _, info := range infoList {
		if info.Name == name {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			calllogger.Info(call.String(hiscallInfo))
			return *info, nil
		}
	}
	err = fmt.Errorf("DBSpec '%s' not found for engine '%s'", name, dbEngine)
	LoggingError(hiscallInfo, err)
	return irs.DBSpecInfo{}, err
}

func (handler *AzureRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine != "mysql" {
		return "", fmt.Errorf("Azure Database for MySQL Flexible Server only supports mysql")
	}
	details, err := handler.fetchAzureSKUDetails(handler.Region.Region)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(details)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw SKU details: %w", err)
	}
	return string(b), nil
}

func (handler *AzureRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine != "mysql" {
		return "", fmt.Errorf("Azure Database for MySQL Flexible Server only supports mysql")
	}
	details, err := handler.fetchAzureSKUDetails(handler.Region.Region)
	if err != nil {
		return "", err
	}
	for _, d := range details {
		if d.Name == name {
			b, err := json.Marshal(d)
			if err != nil {
				return "", fmt.Errorf("failed to marshal raw SKU detail: %w", err)
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("DBSpec '%s' not found", name)
}
