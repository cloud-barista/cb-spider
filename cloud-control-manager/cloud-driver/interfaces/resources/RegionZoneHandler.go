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
	NotApplicable = "N/A"
)

type RegionZoneInfo struct {
	Name        string
	DisplayName string
	ZoneList    []ZoneInfo

	KeyValueList []KeyValue
}

type ZoneInfo struct {
	Name        string
	DisplayName string
	Status      ZoneStatus // Available | Unavailable

	KeyValueList []KeyValue
}

type RegionZoneHandler interface {
	ListRegionZone() ([]*RegionZoneInfo, error)
	ListOrgRegion() (string, error) // return string: json format
	ListOrgZone() (string, error)   // return string: json format
}
