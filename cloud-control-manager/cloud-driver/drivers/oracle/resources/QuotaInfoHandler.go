package resources

import (
	"context"
	"fmt"
	"strconv"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/limits"
)

// OracleQuotaInfoHandler retrieves OCI service limit (quota) information via
// the OCI Limits API (github.com/oracle/oci-go-sdk/v65/limits).
//
// The OCI Limits API always requires the tenancy OCID (root compartment) as
// CompartmentId. The user-defined CompartmentID is used only for
// GetResourceAvailability to reflect actual usage within that compartment.
type OracleQuotaInfoHandler struct {
	Region        idrv.RegionInfo
	TenancyID     string // root compartment = tenancy OCID
	CompartmentID string // user compartment for usage queries
	LimitsClient  limits.LimitsClient
	Ctx           context.Context
}

// ListServiceType returns all OCI service names that have limit definitions.
func (handler *OracleQuotaInfoHandler) ListServiceType() ([]string, error) {
	cblogger.Info("Oracle Driver: called ListServiceType()")

	var serviceNames []string
	pageToken := ""
	pageSize := 100
	for {
		req := limits.ListServicesRequest{
			CompartmentId: common.String(handler.TenancyID),
			Limit:         common.Int(pageSize),
		}
		if pageToken != "" {
			req.Page = common.String(pageToken)
		}
		resp, err := handler.LimitsClient.ListServices(handler.Ctx, req)
		if err != nil {
			return nil, fmt.Errorf("Oracle ListServiceType: %w", err)
		}
		for _, svc := range resp.Items {
			if svc.Name != nil && *svc.Name != "" {
				serviceNames = append(serviceNames, *svc.Name)
			}
		}
		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		pageToken = *resp.OpcNextPage
	}
	return serviceNames, nil
}

// GetQuotaInfo returns all limit values and current usage for the given service.
// It combines:
//  1. ListLimitDefinitions → description + IsResourceAvailabilitySupported flag
//  2. ListLimitValues      → current limit value per scope/AD
//  3. GetResourceAvailability → Used + Available (when supported)
func (handler *OracleQuotaInfoHandler) GetQuotaInfo(serviceType string) (irs.QuotaInfo, error) {
	cblogger.Infof("Oracle Driver: called GetQuotaInfo(serviceType=%s)", serviceType)

	quotaInfo := irs.QuotaInfo{
		CSP:    "ORACLE",
		Region: handler.Region.Region,
	}

	// Step 1: gather limit definitions (description + availability support flag)
	defMap, err := handler.listLimitDefinitions(serviceType)
	if err != nil {
		return quotaInfo, fmt.Errorf("Oracle GetQuotaInfo(%s) ListLimitDefinitions: %w", serviceType, err)
	}

	// Step 2: list limit values (actual quota ceiling per scope)
	limitValues, err := handler.listLimitValues(serviceType)
	if err != nil {
		return quotaInfo, fmt.Errorf("Oracle GetQuotaInfo(%s) ListLimitValues: %w", serviceType, err)
	}

	var quotas []irs.Quota
	for _, lv := range limitValues {
		name := stringValue(lv.Name)
		if name == "" {
			continue
		}

		// Limit value
		limitStr := "NA"
		if lv.Value != nil {
			limitStr = strconv.FormatInt(*lv.Value, 10)
		}

		// Scope suffix for AD-scoped limits
		scopeSuffix := ""
		ad := ""
		if lv.AvailabilityDomain != nil && *lv.AvailabilityDomain != "" {
			ad = *lv.AvailabilityDomain
			scopeSuffix = " [AD:" + ad + "]"
		}

		// Used / Available via GetResourceAvailability
		usedStr := "NA"
		availStr := "NA"
		def, hasDef := defMap[name]
		if hasDef && def.isAvailabilitySupported {
			avail, err := handler.getResourceAvailability(serviceType, name, ad)
			if err == nil {
				if avail.Used != nil {
					usedStr = strconv.FormatInt(*avail.Used, 10)
				}
				if avail.Available != nil {
					availStr = strconv.FormatInt(*avail.Available, 10)
				}
			} else {
				cblogger.Debugf("Oracle GetResourceAvailability(%s/%s): %v", serviceType, name, err)
			}
		}

		desc := ""
		if hasDef {
			desc = def.description
		}

		quotas = append(quotas, irs.Quota{
			QuotaName:   name + scopeSuffix,
			Limit:       limitStr,
			Used:        usedStr,
			Available:   availStr,
			Unit:        "count",
			Description: desc,
		})
	}

	quotaInfo.Quotas = quotas
	return quotaInfo, nil
}

// limitDef holds the fields from LimitDefinitionSummary that we need.
type limitDef struct {
	description             string
	isAvailabilitySupported bool
}

// listLimitDefinitions fetches all limit definitions for the given service and
// returns them as a map keyed by limit name.
func (handler *OracleQuotaInfoHandler) listLimitDefinitions(serviceName string) (map[string]limitDef, error) {
	defs := make(map[string]limitDef)
	pageSize := 100
	pageToken := ""
	for {
		req := limits.ListLimitDefinitionsRequest{
			CompartmentId: common.String(handler.TenancyID),
			ServiceName:   common.String(serviceName),
			Limit:         common.Int(pageSize),
		}
		if pageToken != "" {
			req.Page = common.String(pageToken)
		}
		resp, err := handler.LimitsClient.ListLimitDefinitions(handler.Ctx, req)
		if err != nil {
			return defs, err
		}
		for _, d := range resp.Items {
			if d.Name == nil {
				continue
			}
			isSupported := false
			if d.IsResourceAvailabilitySupported != nil {
				isSupported = *d.IsResourceAvailabilitySupported
			}
			defs[*d.Name] = limitDef{
				description:             stringValue(d.Description),
				isAvailabilitySupported: isSupported,
			}
		}
		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		pageToken = *resp.OpcNextPage
	}
	return defs, nil
}

// listLimitValues fetches all limit values for the given service (all scope types).
func (handler *OracleQuotaInfoHandler) listLimitValues(serviceName string) ([]limits.LimitValueSummary, error) {
	var all []limits.LimitValueSummary
	pageSize := 100
	pageToken := ""
	for {
		req := limits.ListLimitValuesRequest{
			CompartmentId: common.String(handler.TenancyID),
			ServiceName:   common.String(serviceName),
			Limit:         common.Int(pageSize),
		}
		if pageToken != "" {
			req.Page = common.String(pageToken)
		}
		resp, err := handler.LimitsClient.ListLimitValues(handler.Ctx, req)
		if err != nil {
			return all, err
		}
		all = append(all, resp.Items...)
		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		pageToken = *resp.OpcNextPage
	}
	return all, nil
}

// getResourceAvailability calls GetResourceAvailability for a single limit.
// ad should be empty string for REGION/GLOBAL scoped limits.
func (handler *OracleQuotaInfoHandler) getResourceAvailability(serviceName, limitName, ad string) (limits.ResourceAvailability, error) {
	req := limits.GetResourceAvailabilityRequest{
		ServiceName:   common.String(serviceName),
		LimitName:     common.String(limitName),
		CompartmentId: common.String(handler.CompartmentID),
	}
	if ad != "" {
		req.AvailabilityDomain = common.String(ad)
	}
	resp, err := handler.LimitsClient.GetResourceAvailability(handler.Ctx, req)
	if err != nil {
		return limits.ResourceAvailability{}, err
	}
	return resp.ResourceAvailability, nil
}
