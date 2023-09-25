package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/regions"
)

type OpenStackRegionZoneHandler struct {
	Region         idrv.RegionInfo
	IdentityClient *gophercloud.ServiceClient
}

func getZoneList(client *gophercloud.ServiceClient, hiscallInfo call.CLOUDLOGSCHEMA) (*[]irs.ZoneInfo, error) {
	var zoneList []irs.ZoneInfo

	allPages, err := availabilityzones.List(client).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	list, err := availabilityzones.ExtractAvailabilityZones(allPages)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	for _, zone := range list {
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
	}

	return &zoneList, nil
}

func (regionZoneHandler *OpenStackRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.IdentityClient.IdentityEndpoint, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
	start := call.Start()

	listOpts := regions.ListOpts{}
	allPages, err := regions.List(regionZoneHandler.IdentityClient, listOpts).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	regionList, err := regions.ExtractRegions(allPages)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	var regionZoneInfo []*irs.RegionZoneInfo
	var zoneList []irs.ZoneInfo

	for _, region := range regionList {
		client, err := openstack.NewComputeV2(regionZoneHandler.IdentityClient.ProviderClient, gophercloud.EndpointOpts{
			Region: region.ID,
		})
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}

		list, err := getZoneList(client, hiscallInfo)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}

		zoneList = append(zoneList, *list...)

		regionZoneInfo = append(regionZoneInfo, &irs.RegionZoneInfo{
			Name:         region.ID,
			DisplayName:  region.ID,
			ZoneList:     zoneList,
			KeyValueList: []irs.KeyValue{},
		})
	}

	LoggingInfo(hiscallInfo, start)

	return regionZoneInfo, nil
}

func (regionZoneHandler *OpenStackRegionZoneHandler) GetRegionZone(Name string) (irs.RegionZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.IdentityClient.IdentityEndpoint, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
	start := call.Start()

	var zoneList []irs.ZoneInfo

	client, err := openstack.NewComputeV2(regionZoneHandler.IdentityClient.ProviderClient, gophercloud.EndpointOpts{
		Region: Name,
	})
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.RegionZoneInfo{}, getErr
	}

	list, err := getZoneList(client, hiscallInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.RegionZoneInfo{}, getErr
	}

	LoggingInfo(hiscallInfo, start)
	zoneList = append(zoneList, *list...)

	return irs.RegionZoneInfo{
		Name:         Name,
		DisplayName:  Name,
		ZoneList:     zoneList,
		KeyValueList: []irs.KeyValue{},
	}, nil

}

func (regionZoneHandler *OpenStackRegionZoneHandler) ListOrgRegion() (string, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.IdentityClient.IdentityEndpoint, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
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

// ListOrgZone
// Region, Availability Zone 개념이 OpenStack은 다르게 작용함. 현재 구성은 config에 설정된 Region에서 사용가능한 zone 목록을 출력함.
func (regionZoneHandler *OpenStackRegionZoneHandler) ListOrgZone() (string, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.IdentityClient.IdentityEndpoint, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
	start := call.Start()

	client, err := openstack.NewComputeV2(regionZoneHandler.IdentityClient.ProviderClient, gophercloud.EndpointOpts{
		Region: regionZoneHandler.Region.Region,
	})
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	allPages, err := availabilityzones.List(client).AllPages()
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
== GetRegionZone 실행 예시 ==
[CLOUD-BARISTA].[INFO]: 2023-09-25 15:37:35 Test_Resources.go:1144, main.testRegionZoneHandler() - Start GetRegionZone() ...
Enter Region Name: RegionOne
[CLOUD-BARISTA].[INFO]: 2023-09-25 15:37:40 CommonOpenStackFunc.go:52, github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/resources.GetCallLogScheme() - Call OPENSTACK ListOrgRegion()
[HISCALL].[124.53.55.55] 2023-09-25 15:37:40 (Monday) github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/resources.LoggingInfo():48 - "CloudOS" : "OPENSTACK", "RegionZone" : "http://192.168.110.170:5000/v3/", "ResourceType" : "REGIONZONE", "ResourceName" : "RegionZone", "CloudOSAPI" : "ListOrgRegion()", "ElapsedTime" : "0.2171", "ErrorMSG" : ""
(resources.RegionZoneInfo) {
 Name: (string) (len=9) "RegionOne",
 DisplayName: (string) (len=9) "RegionOne",
 ZoneList: ([]resources.ZoneInfo) (len=1 cap=1) {
  (resources.ZoneInfo) {
   Name: (string) (len=4) "nova",
   DisplayName: (string) (len=4) "nova",
   Status: (resources.ZoneStatus) (len=9) "Available",
   KeyValueList: ([]resources.KeyValue) {
   }
  }
 },
 KeyValueList: ([]resources.KeyValue) {
 }
}
[CLOUD-BARISTA].[INFO]: 2023-09-25 15:37:40 Test_Resources.go:1155, main.testRegionZoneHandler() - Finish GetRegionZone()
*/

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
