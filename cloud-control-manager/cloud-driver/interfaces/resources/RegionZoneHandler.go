// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2023.09.

package resources

// -------- Const
type ZoneStatus string

const (
	ZoneAvailable   ZoneStatus = "Available"
	ZoneUnavailable ZoneStatus = "Unavailable"
	NotSupported    ZoneStatus = "StatusNotSupported"
)

// RegionZoneInfo represents the information of a Region Zone.
// @example {"Name": "us-east", "DisplayName": "United States, Ohio", "CSPDisplayName":"US East (N. Virginia)", "ZoneList": [{"Name": "us-east-1a", "DisplayName": "United States, Ohio", "CSPDisplayName":"US East (N. Virginia)", "Status": "Available"}], "KeyValueList": [{"Key": "regionKey1", "Value": "regionValue1"}]}
type RegionZoneInfo struct {
	Name           string     `json:"Name" validate:"required" example:"us-east"`
	DisplayName    string     `json:"DisplayName" validate:"required" example:"United States, Ohio"`
	CSPDisplayName string     `json:"CSPDisplayName" validate:"required" example:"US East (N. Virginia)"`
	ZoneList       []ZoneInfo `json:"ZoneList,omitempty" validate:"omitempty"`
	KeyValueList   []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// ZoneInfo represents the information of a Zone.
// @example {"Name": "us-east-1a", "DisplayName": "United States, Ohio", "CSPDisplayName":"US East (N. Virginia)", "Status": "Available", "KeyValueList": [{"Key": "zoneKey1", "Value": "zoneValue1"}]}
type ZoneInfo struct {
	Name           string     `json:"Name" validate:"required" example:"us-east-1a"`
	DisplayName    string     `json:"DisplayName" validate:"required" example:"United States, Ohio"`
	CSPDisplayName string     `json:"CSPDisplayName" validate:"required" example:"US East (N. Virginia)"`
	Status         ZoneStatus `json:"Status" validate:"required" example:"Available"`
	KeyValueList   []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

type RegionZoneHandler interface {
	ListRegionZone() ([]*RegionZoneInfo, error)
	GetRegionZone(Name string) (RegionZoneInfo, error)

	ListOrgRegion() (string, error) // return string: json format
	ListOrgZone() (string, error)   // return string: json format
}
