// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2023.09.

package resources

import (
	"encoding/json"
	"fmt"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var regionZoneInfoMap map[string][]*irs.RegionZoneInfo

type MockRegionZoneHandler struct {
	Region   idrv.RegionInfo
	MockName string
}

var prepareRegionZoneInfoList []*irs.RegionZoneInfo

func init() {
	regionZoneInfoMap = make(map[string][]*irs.RegionZoneInfo)
}

// Be called before using the User function.
// Called in MockDriver
func PrepareRegionZone(mockName string) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called prepare()!")

	if regionZoneInfoMap[mockName] != nil {
		return
	}

	prepareRegionZoneInfoList = []*irs.RegionZoneInfo{
		{
			Name:        "mercury",
			DisplayName: "Mercury Region",
			ZoneList: []irs.ZoneInfo{
				{
					Name:         "mercury-z1",
					DisplayName:  "Mercury Zone 1",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "mercury-z2",
					DisplayName:  "Mercury Zone 2",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "mercury-z3",
					DisplayName:  "Mercury Zone 3",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
			},
			KeyValueList: nil,
		},
		{
			Name:        "venus",
			DisplayName: "Venus Region",
			ZoneList: []irs.ZoneInfo{
				{
					Name:         "venus-z1",
					DisplayName:  "Venus Zone 1",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "venus-z2",
					DisplayName:  "Venus Zone 2",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "venus-z3",
					DisplayName:  "Venus Zone 3",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
			},
			KeyValueList: nil,
		},
		{
			Name:        "mars",
			DisplayName: "Mars Region",
			ZoneList: []irs.ZoneInfo{
				{
					Name:         "mars-z1",
					DisplayName:  "Mars Zone 1",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "mars-z2",
					DisplayName:  "Mars Zone 2",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "mars-z3",
					DisplayName:  "Mars Zone 3",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
			},
			KeyValueList: nil,
		},
		{
			Name:        "jupiter",
			DisplayName: "Jupiter Region",
			ZoneList: []irs.ZoneInfo{
				{
					Name:         "jupiter-z1",
					DisplayName:  "Jupiter Zone 1",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "jupiter-z2",
					DisplayName:  "Jupiter Zone 2",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "jupiter-z3",
					DisplayName:  "Jupiter Zone 3",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
			},
			KeyValueList: nil,
		},
		{
			Name:        "saturn",
			DisplayName: "Saturn Region",
			ZoneList: []irs.ZoneInfo{
				{
					Name:         "saturn-z1",
					DisplayName:  "Saturn Zone 1",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "saturn-z2",
					DisplayName:  "Saturn Zone 2",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "saturn-z3",
					DisplayName:  "Saturn Zone 3",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
			},
			KeyValueList: nil,
		},
		{
			Name:        "uranus",
			DisplayName: "Uranus Region",
			ZoneList: []irs.ZoneInfo{
				{
					Name:         "uranus-z1",
					DisplayName:  "Uranus Zone 1",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "uranus-z2",
					DisplayName:  "Uranus Zone 2",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "uranus-z3",
					DisplayName:  "Uranus Zone 3",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
			},
			KeyValueList: nil,
		},
		{
			Name:        "neptune",
			DisplayName: "Neptune Region",
			ZoneList: []irs.ZoneInfo{
				{
					Name:         "neptune-z1",
					DisplayName:  "Neptune Zone 1",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "neptune-z2",
					DisplayName:  "Neptune Zone 2",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
				{
					Name:         "neptune-z3",
					DisplayName:  "Neptune Zone 3",
					Status:       irs.ZoneAvailable,
					KeyValueList: nil,
				},
			},
			KeyValueList: nil,
		},
	}
	regionZoneInfoMap[mockName] = prepareRegionZoneInfoList
}

func (handler *MockRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListRegionZone()!")

	mockName := handler.MockName

	infoList, ok := regionZoneInfoMap[mockName]
	if !ok {
		return []*irs.RegionZoneInfo{}, nil
	}

	// cloning list of RegionZoneInfo
	resultList := make([]*irs.RegionZoneInfo, len(infoList))
	copy(resultList, infoList)
	return resultList, nil
}

func (handler *MockRegionZoneHandler) GetRegionZone(Name string) (irs.RegionZoneInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetRegionZone()!")

	infoList, err := handler.ListRegionZone()
	if err != nil {
		cblogger.Error(err)
		return irs.RegionZoneInfo{}, err
	}

	for _, info := range infoList {
		if (*info).Name == Name {
			return *info, nil
		}
	}

	return irs.RegionZoneInfo{}, fmt.Errorf("%s Name does not exist!!", Name)
}

// ListOrgRegion implements resources.RegionZoneHandler.
func (handler *MockRegionZoneHandler) ListOrgRegion() (string, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListOrgRegion()!")

	// Convert prepareRegionZoneInfoList to JSON
	jsonData, err := json.MarshalIndent(prepareRegionZoneInfoList, "", "  ")
	if err != nil {
		cblogger.Error("Error while converting to JSON: ", err)
		return "", err
	}

	return string(jsonData), nil
}

// ListOrgZone implements resources.RegionZoneHandler.
func (handler *MockRegionZoneHandler) ListOrgZone() (string, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListOrgZone()!")

	for _, info := range prepareRegionZoneInfoList {
		if (*info).Name == handler.Region.Region {
			jsonData, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				cblogger.Error("Error while converting to JSON: ", err)
				return "", err
			}
			return string(jsonData), nil
		}
	}

	return "", fmt.Errorf("%s The original zone list does not exist!!", handler.Region.Region)
}
