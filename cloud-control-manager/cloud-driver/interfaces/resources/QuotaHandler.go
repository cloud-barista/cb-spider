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

// ResourceQuota holds the quota (limit), current usage, and availability
// for a single resource type in a CSP region.
// Fields that the CSP does not expose are set to the string "NA".
type ResourceQuota struct {
	// ResourceType is a canonical name for the resource category,
	// e.g. "VM", "vCPU", "VPC", "Subnet", "SecurityGroup", …
	ResourceType string `json:"ResourceType" validate:"required" example:"VM"`

	// Limit is the maximum number (or size) allowed by the CSP quota.
	// "NA" when the CSP does not expose this value via API.
	Limit string `json:"Limit" validate:"required" example:"500"`

	// Used is the amount currently consumed.
	// "NA" when the CSP does not expose this value via API.
	Used string `json:"Used" validate:"required" example:"12"`

	// Available is the headroom (Limit - Used).
	// "NA" when either Limit or Used is "NA".
	Available string `json:"Available" validate:"required" example:"488"`

	// Unit describes the dimension being counted, e.g. "count", "vCPU", "GB".
	Unit string `json:"Unit" validate:"required" example:"count"`

	// Description is a human-readable explanation of the quota item.
	Description string `json:"Description,omitempty" example:"Maximum number of running VM instances"`
}

// QuotaInfo aggregates all resource quotas for a given connection / region.
type QuotaInfo struct {
	// CSP is the cloud provider name, e.g. "AWS", "Azure", "GCP", "Alibaba", "IBM".
	CSP string `json:"CSP" validate:"required" example:"AWS"`

	// Region is the region (or location) the quotas apply to.
	Region string `json:"Region" validate:"required" example:"us-east-1"`

	// ResourceQuotas lists per-resource quota details.
	ResourceQuotas []ResourceQuota `json:"ResourceQuotas" validate:"required"`

	// KeyValueList carries additional CSP-specific quota metadata.
	KeyValueList []KeyValue `json:"KeyValueList,omitempty"`
}

// QuotaHandler defines the interface for retrieving CSP resource quotas.
// Drivers that support quota inspection must implement this interface and
// advertise the capability via DriverCapabilityInfo.QuotaHandler = true.
type QuotaHandler interface {
	// GetQuota returns the current quota limits and usage for key compute
	// resources (VM, vCPU, VPC, Subnet, SecurityGroup, Disk, NLB, PublicIP,
	// KeyPair, …) in the configured region.
	//
	// Values the CSP does not expose through its API are represented as "NA".
	GetQuota() (QuotaInfo, error)
}
