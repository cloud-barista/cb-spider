// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resources interfaces of Cloud Driver.
//
// by CB-Spider Team, 2026.03.

package resources

// Quota holds the quota (limit), current usage, and availability
// for a single quota item in a CSP region.
// Fields that the CSP does not expose are set to the string "NA".
type Quota struct {
	// QuotaName is the CSP's original quota name or code as returned by the API.
	QuotaName string `json:"QuotaName" validate:"required" example:"vm-instances"`

	// Limit is the maximum number (or size) allowed by the CSP quota.
	// "NA" when the CSP does not expose this value via API.
	Limit string `json:"Limit" validate:"required" example:"500"`

	// Used is the amount currently consumed.
	// "NA" when the CSP does not expose this value via API.
	Used string `json:"Used" validate:"required" example:"12"`

	// Available is the remaining capacity (Limit - Used).
	// "NA" when either Limit or Used is "NA".
	Available string `json:"Available" validate:"required" example:"488"`

	// Unit describes the dimension being counted, e.g. "count", "vCPU", "GB".
	// "NA" when the CSP does not expose this value via API.
	Unit string `json:"Unit" validate:"required" example:"count"`

	// Description is an optional human-readable explanation of the quota item,
	// passed through from the CSP as-is.
	Description string `json:"Description,omitempty"`
}

// QuotaInfo aggregates all quotas for a given connection / region.
type QuotaInfo struct {
	// CSP is the cloud provider name, e.g. "AWS", "Azure", "GCP", "Alibaba", "IBM".
	CSP string `json:"CSP" validate:"required" example:"AWS"`

	// Region is the region (or location) the quotas apply to.
	Region string `json:"Region" validate:"required" example:"us-east-1"`

	// Quotas lists quota details per resource.
	Quotas []Quota `json:"Quotas" validate:"required"`

	// KeyValueList carries additional CSP-specific quota metadata.
	KeyValueList []KeyValue `json:"KeyValueList,omitempty"`
}

// QuotaInfoHandler defines the interface for retrieving CSP quota info.
// Drivers that support quota inspection must implement this interface and
// advertise the capability via DriverCapabilityInfo.QuotaInfoHandler = true.
//
// The workflow is a two-step process:
//  1. Call ListServiceType() to discover available service categories.
//  2. Call GetQuotaInfo(serviceType) to retrieve all quotas under that service type.
type QuotaInfoHandler interface {
	// ListServiceType returns the list of service type names (product/service
	// categories) for which quota information is available.
	// For example, AWS returns service codes such as "ec2", "vpc", "ebs", etc.
	ListServiceType() ([]string, error)

	// GetQuotaInfo returns the quota limits and usage for ALL quota items belonging
	// to the given service type. No filtering or name-mapping is performed;
	// the CSP's original quota names & values are passed through as-is.
	//
	// Values the CSP does not expose through its API are represented as "NA".
	GetQuotaInfo(serviceType string) (QuotaInfo, error)
}
