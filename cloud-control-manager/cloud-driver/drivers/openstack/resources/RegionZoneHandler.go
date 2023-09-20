package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/regions"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OpenStackRegionZoneHandler struct {
	IdentityClient *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
}

// Region, Availability Zone 개념이 OpenStack은 다르게 작용함. Region별 Availability Zone 가져오는 API 제공 안됨.
func (regionZoneHandler *OpenStackRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.ComputeClient.IdentityEndpoint, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
	start := call.Start()

	allPages, err := availabilityzones.List(regionZoneHandler.ComputeClient).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	zoneList, err := availabilityzones.ExtractAvailabilityZones(allPages)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	LoggingInfo(hiscallInfo, start)

	var regionZoneInfo []*irs.RegionZoneInfo

	for _, zone := range zoneList {
		var zoneList []irs.ZoneInfo
		var status irs.ZoneStatus

		if zone.ZoneState.Available {
			status = irs.ZoneAvailable
		} else {
			status = irs.ZoneUnavailable
		}

		zoneList = append(zoneList, irs.ZoneInfo{
			Name:         zone.ZoneName,
			DisplayName:  zone.ZoneName,
			Status:       status,
			KeyValueList: []irs.KeyValue{},
		})

		regionZoneInfo = append(regionZoneInfo, &irs.RegionZoneInfo{
			Name:         "N/A",
			DisplayName:  "N/A",
			ZoneList:     zoneList,
			KeyValueList: []irs.KeyValue{},
		})
	}

	return regionZoneInfo, nil
}

// Region, Availability Zone 개념이 OpenStack은 다르게 작용함. Region별 Availability Zone 가져오는 API 제공 안됨.
func (regionZoneHandler *OpenStackRegionZoneHandler) GetRegionZone(Name string) (irs.RegionZoneInfo, error) {
	return irs.RegionZoneInfo{}, errors.New("Driver: not implemented")
}

func (regionZoneHandler *OpenStackRegionZoneHandler) ListOrgRegion() (string, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.ComputeClient.IdentityEndpoint, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
	start := call.Start()

	listOpts := regions.ListOpts{}
	allPages, err := regions.List(regionZoneHandler.IdentityClient, listOpts).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgRegion. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	regionList, err := regions.ExtractRegions(allPages)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgRegion. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	LoggingInfo(hiscallInfo, start)

	jsonBytes, err := json.Marshal(regionList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgRegion. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

func (regionZoneHandler *OpenStackRegionZoneHandler) ListOrgZone() (string, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.ComputeClient.IdentityEndpoint, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
	start := call.Start()

	allPages, err := availabilityzones.List(regionZoneHandler.ComputeClient).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	zoneList, err := availabilityzones.ExtractAvailabilityZones(allPages)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	LoggingInfo(hiscallInfo, start)

	jsonBytes, err := json.Marshal(zoneList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

/*
== ListOrgRegion()  결과 값 예시 ==
[
  {
    "description": "",
    "id": "RegionOne",
    "links": {
      "self": "http://192.168.110.170:5000/v3/regions/RegionOne"
    },
    "parent_region_id": ""
  }
]
*/

/*
== ListOrgZone()  결과 값 예시 ==
[
  {
    "hosts": null,
    "zoneName": "nova",
    "zoneState": {
      "available": true
    }
  }
]
*/
