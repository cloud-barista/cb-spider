// Azure Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Azure Driver.
//
// by CB-Spider Team, 2025.07.

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v3"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureQuotaInfoHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
}

// azureUsageResponse represents the generic Azure usage REST API response.
type azureUsageResponse struct {
	Value []azureUsageItem `json:"value"`
}

// azureUsageItem represents one usage entry from the Azure REST API.
type azureUsageItem struct {
	Name         azureUsageName `json:"name"`
	CurrentValue *float64       `json:"currentValue"`
	Limit        *float64       `json:"limit"`
	Unit         string         `json:"unit"`
}

// azureUsageName holds the internal and localized display name.
// Implements custom UnmarshalJSON to handle both object and plain-string formats.
type azureUsageName struct {
	Value          string
	LocalizedValue string
}

// UnmarshalJSON handles Azure's inconsistent name formats:
//   - Object: {"value": "...", "localizedValue": "..."}
//   - String: "someName"
func (n *azureUsageName) UnmarshalJSON(data []byte) error {
	// Try object format first
	var obj struct {
		Value          string `json:"value"`
		LocalizedValue string `json:"localizedValue"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		n.Value = obj.Value
		n.LocalizedValue = obj.LocalizedValue
		return nil
	}
	// Fallback: plain string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n.Value = s
		n.LocalizedValue = s
		return nil
	}
	return fmt.Errorf("azureUsageName: cannot unmarshal %s", string(data))
}

// azureSkipProviders lists providers whose usage API is globally broken/retired,
// despite still being registered. These are excluded from ListServiceType results.
//   - DataLakeAnalytics: globally retired (2024-02-29), DNS no longer resolves
//   - DataLakeStore: globally retired (2024-02-29), DNS no longer resolves
//   - AVS (Azure VMware Solution): usage API rejects all API versions globally
//     (HTTP 400 "Unrecognized API version") — confirmed in both koreacentral and eastus
var azureSkipProviders = map[string]bool{
	"microsoft.datalakeanalytics": true,
	"microsoft.datalakestore":     true,
	"microsoft.avs":               true,
}

// newCredential creates a ClientSecretCredential from the handler's credential info.
func (handler *AzureQuotaInfoHandler) newCredential() (*azidentity.ClientSecretCredential, error) {
	return azidentity.NewClientSecretCredential(
		handler.CredentialInfo.TenantId,
		handler.CredentialInfo.ClientId,
		handler.CredentialInfo.ClientSecret,
		nil,
	)
}

// ListServiceType dynamically discovers Azure resource providers that expose
// usage/quota APIs by checking for the "locations/usages" resource type
// among registered providers.
func (handler *AzureQuotaInfoHandler) ListServiceType() ([]string, error) {
	cblogger.Info("Azure Driver: called ListServiceType()")

	cred, err := handler.newCredential()
	if err != nil {
		return nil, err
	}

	subscriptionID := handler.CredentialInfo.SubscriptionId
	providersClient, err := armresources.NewProvidersClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("Azure ListServiceType: failed to create providers client: %w", err)
	}

	var serviceTypes []string
	pager := providersClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(handler.Ctx)
		if err != nil {
			return nil, fmt.Errorf("Azure ListServiceType: %w", err)
		}
		for _, provider := range page.Value {
			if provider.Namespace == nil || provider.RegistrationState == nil {
				continue
			}
			// Only include providers registered in this subscription
			if *provider.RegistrationState != "Registered" {
				continue
			}
			// Skip providers with known-broken usage APIs
			if azureSkipProviders[strings.ToLower(*provider.Namespace)] {
				continue
			}
			// Check if this provider exposes "locations/usages"
			for _, rt := range provider.ResourceTypes {
				if rt.ResourceType != nil && strings.EqualFold(*rt.ResourceType, "locations/usages") {
					// Verify this provider supports our region
					if !handler.isRegionSupported(rt.Locations) {
						break
					}
					name := *provider.Namespace
					// Strip "Microsoft." prefix for a shorter, user-friendly name
					name = strings.TrimPrefix(name, "Microsoft.")
					serviceTypes = append(serviceTypes, name)
					break
				}
			}
		}
	}

	sort.Strings(serviceTypes)
	return serviceTypes, nil
}

// normalizeLocation converts a location name to a normalized form for comparison.
// E.g., "Korea Central" → "koreacentral", "koreacentral" → "koreacentral".
func normalizeLocation(loc string) string {
	return strings.ToLower(strings.ReplaceAll(loc, " ", ""))
}

// isRegionSupported checks if the handler's region is in the given locations list.
// If the locations list is empty, it is assumed all regions are supported.
func (handler *AzureQuotaInfoHandler) isRegionSupported(locations []*string) bool {
	if len(locations) == 0 {
		return true
	}
	target := normalizeLocation(handler.Region.Region)
	for _, loc := range locations {
		if loc != nil && normalizeLocation(*loc) == target {
			return true
		}
	}
	return false
}

// GetQuotaInfo retrieves ALL usage/quota items for the given Azure resource provider
// using the generic REST API pattern: /subscriptions/{sub}/providers/{ns}/locations/{loc}/usages.
// No filtering or name-mapping is performed; CSP-original values are passed through.
func (handler *AzureQuotaInfoHandler) GetQuotaInfo(serviceType string) (irs.QuotaInfo, error) {
	cblogger.Infof("Azure Driver: called GetQuotaInfo(serviceType=%s)", serviceType)

	quotaInfo := irs.QuotaInfo{
		CSP:    "AZURE",
		Region: handler.Region.Region,
	}

	cred, err := handler.newCredential()
	if err != nil {
		return quotaInfo, err
	}

	subscriptionID := handler.CredentialInfo.SubscriptionId
	location := handler.Region.Region

	// Build the full provider namespace (e.g. "Compute" → "Microsoft.Compute")
	namespace := serviceType
	if !strings.Contains(namespace, ".") {
		namespace = "Microsoft." + namespace
	}

	// Look up the API version for "locations/usages" of this provider
	apiVersion, err := handler.getUsageAPIVersion(cred, subscriptionID, namespace)
	if err != nil {
		return quotaInfo, fmt.Errorf("Azure GetQuotaInfo: failed to get API version for %s: %w", namespace, err)
	}

	token, err := cred.GetToken(handler.Ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return quotaInfo, fmt.Errorf("Azure GetQuotaInfo: failed to get token: %w", err)
	}

	// Call the usage REST API; if the API version is not supported for this
	// region, parse the supported versions from the error and retry once.
	body, err := handler.callUsageAPI(subscriptionID, namespace, location, apiVersion, token.Token)
	if err != nil {
		// Check if error contains supported API versions list
		fallbackVer := extractLatestSupportedVersion(err.Error())
		if fallbackVer != "" && fallbackVer != apiVersion {
			cblogger.Infof("Azure GetQuotaInfo: retrying %s with fallback API version %s", namespace, fallbackVer)
			body, err = handler.callUsageAPI(subscriptionID, namespace, location, fallbackVer, token.Token)
		}
		if err != nil {
			return quotaInfo, err
		}
	}

	var usageResp azureUsageResponse
	if err := json.Unmarshal(body, &usageResp); err != nil {
		// Some providers return a bare array instead of {"value": [...]}
		var items []azureUsageItem
		if err2 := json.Unmarshal(body, &items); err2 != nil {
			return quotaInfo, fmt.Errorf("Azure GetQuotaInfo: failed to parse response: %w", err)
		}
		usageResp.Value = items
	}

	var quotas []irs.Quota
	for _, item := range usageResp.Value {
		nameVal := item.Name.Value
		localName := item.Name.LocalizedValue
		if localName == "" {
			localName = nameVal
		}

		limit := "NA"
		used := "NA"
		available := "NA"

		if item.Limit != nil {
			limit = strconv.FormatFloat(*item.Limit, 'f', -1, 64)
		}
		if item.CurrentValue != nil {
			used = strconv.FormatFloat(*item.CurrentValue, 'f', -1, 64)
		}
		if item.Limit != nil && item.CurrentValue != nil {
			av := *item.Limit - *item.CurrentValue
			available = strconv.FormatFloat(av, 'f', -1, 64)
		}

		unit := "NA"
		if item.Unit != "" {
			unit = item.Unit
		}

		rq := irs.Quota{
			QuotaName:   nameVal,
			Limit:       limit,
			Used:        used,
			Available:   available,
			Unit:        unit,
			Description: fmt.Sprintf("%s", localName),
		}
		quotas = append(quotas, rq)
	}

	quotaInfo.Quotas = quotas
	return quotaInfo, nil
}

// getUsageAPIVersion looks up the latest API version for the "locations/usages"
// resource type of the given provider namespace.
func (handler *AzureQuotaInfoHandler) getUsageAPIVersion(
	cred *azidentity.ClientSecretCredential,
	subscriptionID, namespace string,
) (string, error) {
	providersClient, err := armresources.NewProvidersClient(subscriptionID, cred, nil)
	if err != nil {
		return "", err
	}

	resp, err := providersClient.Get(handler.Ctx, namespace, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get provider %s: %w", namespace, err)
	}

	for _, rt := range resp.ResourceTypes {
		if rt.ResourceType != nil && strings.EqualFold(*rt.ResourceType, "locations/usages") {
			// Prefer stable (non-preview) API versions over preview versions
			var stableVersion, previewVersion string
			for _, v := range rt.APIVersions {
				if v == nil {
					continue
				}
				if strings.Contains(strings.ToLower(*v), "preview") {
					if previewVersion == "" {
						previewVersion = *v
					}
				} else {
					if stableVersion == "" {
						stableVersion = *v
					}
				}
			}
			if stableVersion != "" {
				return stableVersion, nil
			}
			if previewVersion != "" {
				return previewVersion, nil
			}
		}
	}

	return "", fmt.Errorf("no API version found for %s/locations/usages", namespace)
}

// callUsageAPI makes an HTTP GET to the Azure usage REST API and returns the
// response body on success. Returns an error (including the response body) on
// non-200 status codes.
func (handler *AzureQuotaInfoHandler) callUsageAPI(subscriptionID, namespace, location, apiVersion, bearerToken string) ([]byte, error) {
	usageURL := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/providers/%s/locations/%s/usages?api-version=%s",
		subscriptionID, namespace, location, apiVersion,
	)

	req, err := http.NewRequestWithContext(handler.Ctx, http.MethodGet, usageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Azure GetQuotaInfo: REST call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		serviceName := strings.TrimPrefix(namespace, "Microsoft.")
		switch {
		case resp.StatusCode == 502:
			return nil, fmt.Errorf("Azure GetQuotaInfo(%s): service endpoint unavailable (HTTP 502). "+
				"This service may be retired or not available in region '%s'", serviceName, location)
		case resp.StatusCode == 400:
			return nil, fmt.Errorf("Azure GetQuotaInfo(%s): bad request (HTTP 400) in region '%s'. "+
				"The service may not support usage/quota API in this region. Detail: %s",
				serviceName, location, string(body))
		case resp.StatusCode == 404:
			return nil, fmt.Errorf("Azure GetQuotaInfo(%s): usage API not found (HTTP 404) in region '%s'. "+
				"This service may not be available in this region", serviceName, location)
		default:
			return nil, fmt.Errorf("Azure GetQuotaInfo(%s): HTTP %d in region '%s': %s",
				serviceName, resp.StatusCode, location, string(body))
		}
	}

	return body, nil
}

// supportedVersionsRegex captures the portion after "supported api-versions are".
var supportedVersionsRegex = regexp.MustCompile(`(?i)supported api-versions are\s*'([^']+)'`)

// apiVersionRegex matches API version strings like "2024-07-01" or "2024-07-01-preview".
var apiVersionRegex = regexp.MustCompile(`\d{4}-\d{2}-\d{2}(?:-preview)?`)

// extractLatestSupportedVersion parses the Azure error message to find the
// latest non-preview API version from the "supported api-versions" list.
// Only versions after "supported api-versions are" are considered, so the
// originally-requested version in the error message is not picked up.
// Returns empty string if no versions can be parsed.
func extractLatestSupportedVersion(errMsg string) string {
	// Extract only the supported versions list portion
	match := supportedVersionsRegex.FindStringSubmatch(errMsg)
	if len(match) < 2 {
		return ""
	}
	versionList := match[1]

	versions := apiVersionRegex.FindAllString(versionList, -1)
	if len(versions) == 0 {
		return ""
	}

	// Pick the latest stable (non-preview) version
	var latestStable, latestPreview string
	for _, v := range versions {
		if strings.Contains(v, "preview") {
			if v > latestPreview {
				latestPreview = v
			}
		} else {
			if v > latestStable {
				latestStable = v
			}
		}
	}
	if latestStable != "" {
		return latestStable
	}
	return latestPreview
}
