package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"sync"
)

type IbmRegionZoneHandler struct {
	Region     idrv.RegionInfo
	VpcService *vpcv1.VpcV1
	Ctx        context.Context
}

func (regionZoneHandler *IbmRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.Region, call.REGIONZONE, "RegionZone", "ListRegionZone()")
	start := call.Start()

	options := &vpcv1.ListRegionsOptions{}
	regions, _, err := regionZoneHandler.VpcService.ListRegionsWithContext(regionZoneHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgRegion. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	var regionZoneInfo []*irs.RegionZoneInfo

	var routineMax = 20
	var wait sync.WaitGroup
	var mutex = &sync.Mutex{}
	var lenRegions = len(regions.Regions)
	var zoneErrorOccurred bool

	for i := 0; i < lenRegions; {
		if lenRegions-i < routineMax {
			routineMax = lenRegions - i
		}

		wait.Add(routineMax)

		for j := 0; j < routineMax; j++ {
			go func(wait *sync.WaitGroup, reg vpcv1.Region) {
				var zoneList []irs.ZoneInfo

				options := &vpcv1.ListRegionZonesOptions{
					RegionName: reg.Name,
				}
				zones, _, err := regionZoneHandler.VpcService.ListRegionZonesWithContext(regionZoneHandler.Ctx, options)
				if err != nil {
					zoneErrorOccurred = true
					return
				}

				for _, zone := range zones.Zones {
					var status = irs.ZoneAvailable

					if *zone.Status != vpcv1.ZoneStatusAvailableConst {
						status = irs.ZoneUnavailable
					}
					zoneList = append(zoneList, irs.ZoneInfo{
						Name:         *zone.Name,
						DisplayName:  *zone.Name,
						Status:       status,
						KeyValueList: []irs.KeyValue{},
					})
				}

				mutex.Lock()
				regionZoneInfo = append(regionZoneInfo, &irs.RegionZoneInfo{
					Name:         *reg.Name,
					DisplayName:  *reg.Name,
					ZoneList:     zoneList,
					KeyValueList: []irs.KeyValue{},
				})
				mutex.Unlock()

				wait.Done()
			}(&wait, regions.Regions[j])

			i++
			if i == lenRegions {
				break
			}
		}

		wait.Wait()
	}

	if zoneErrorOccurred {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s",
			"Error occurred while getting zone info."))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return regionZoneInfo, nil
}

func (regionZoneHandler *IbmRegionZoneHandler) GetRegionZone(Name string) (irs.RegionZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.Region, call.REGIONZONE, "RegionZone", "GetRegionZone()")
	start := call.Start()

	var zoneList []irs.ZoneInfo

	options := &vpcv1.ListRegionZonesOptions{
		RegionName: &Name,
	}
	zones, _, err := regionZoneHandler.VpcService.ListRegionZonesWithContext(regionZoneHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.RegionZoneInfo{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	for _, zone := range zones.Zones {
		var status = irs.ZoneAvailable

		if *zone.Status != vpcv1.ZoneStatusAvailableConst {
			status = irs.ZoneUnavailable
		}
		zoneList = append(zoneList, irs.ZoneInfo{
			Name:         *zone.Name,
			DisplayName:  *zone.Name,
			Status:       status,
			KeyValueList: []irs.KeyValue{},
		})
	}

	return irs.RegionZoneInfo{
		Name:         Name,
		DisplayName:  Name,
		ZoneList:     zoneList,
		KeyValueList: []irs.KeyValue{},
	}, nil
}

func (regionZoneHandler *IbmRegionZoneHandler) ListOrgRegion() (string, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.Region, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
	start := call.Start()

	options := &vpcv1.ListRegionsOptions{}
	regions, _, err := regionZoneHandler.VpcService.ListRegionsWithContext(regionZoneHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgRegion. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	LoggingInfo(hiscallInfo, start)

	type region struct {
		// The API endpoint for this region.
		Endpoint *string `json:"endpoint" validate:"required"`
		// The URL for this region.
		Href *string `json:"href" validate:"required"`
		// The globally unique name for this region.
		Name *string `json:"name" validate:"required"`
		// The availability status of this region.
		Status *string `json:"status" validate:"required"`
	}

	var regionList struct {
		List []region `json:"list"`
	}

	for _, reg := range regions.Regions {
		regionList.List = append(regionList.List, region{
			Endpoint: reg.Endpoint,
			Href:     reg.Href,
			Name:     reg.Name,
			Status:   reg.Status,
		})
	}

	jsonBytes, err := json.Marshal(regionList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgRegion. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

func (regionZoneHandler *IbmRegionZoneHandler) ListOrgZone() (string, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.Region, call.REGIONZONE, "RegionZone", "ListOrgZone()")
	start := call.Start()

	options := &vpcv1.ListRegionZonesOptions{
		RegionName: &regionZoneHandler.Region.Region,
	}
	zones, _, err := regionZoneHandler.VpcService.ListRegionZonesWithContext(regionZoneHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	LoggingInfo(hiscallInfo, start)

	// RegionReference : RegionReference struct
	type regionReference struct {
		// The URL for this region.
		Href string `json:"href" validate:"required"`
		// The globally unique name for this region.
		Name string `json:"name" validate:"required"`
	}

	type zone struct {
		// The URL for this zone.
		Href string `json:"href" validate:"required"`
		// The globally unique name for this zone.
		Name string `json:"name" validate:"required"`
		// The region this zone resides in.
		Region regionReference `json:"region" validate:"required"`
		// The availability status of this zone.
		Status string `json:"status" validate:"required"`
	}

	var zoneList struct {
		List []zone `json:"list"`
	}

	for _, zo := range zones.Zones {
		zoneList.List = append(zoneList.List, zone{
			Href: *zo.Href,
			Name: *zo.Name,
			Region: regionReference{
				Href: *zo.Region.Href,
				Name: *zo.Region.Name,
			},
			Status: *zo.Status,
		})
	}

	jsonBytes, err := json.Marshal(zoneList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgZone. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

/*
== GetRegionZone 실행 예시 ==
[CLOUD-BARISTA].[INFO]: 2023-09-26 15:20:27 Test_Resources.go:1404, main.testRegionZoneHandler() - Start GetRegionZone() ...
Enter Region Name: us-south
[CLOUD-BARISTA].[INFO]: 2023-09-26 15:20:32 CommonIbmFunc.go:43, github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/resources.GetCallLogScheme() - Call IBM GetRegionZone()
[HISCALL].[124.53.55.55] 2023-09-26 15:20:33 (Tuesday) github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/resources.LoggingInfo():39 - "CloudOS" : "IBM", "RegionZone" : "us-south", "ResourceType" : "REGIONZONE", "ResourceName" : "RegionZone", "CloudOSAPI" : "GetRegionZone()", "ElapsedTime" : "0.8174", "ErrorMSG" : ""
(resources.RegionZoneInfo) {
 Name: (string) (len=8) "us-south",
 DisplayName: (string) (len=8) "us-south",
 ZoneList: ([]resources.ZoneInfo) (len=3 cap=4) {
  (resources.ZoneInfo) {
   Name: (string) (len=10) "us-south-1",
   DisplayName: (string) (len=10) "us-south-1",
   Status: (resources.ZoneStatus) (len=9) "Available",
   KeyValueList: ([]resources.KeyValue) {
   }
  },
  (resources.ZoneInfo) {
   Name: (string) (len=10) "us-south-2",
   DisplayName: (string) (len=10) "us-south-2",
   Status: (resources.ZoneStatus) (len=9) "Available",
   KeyValueList: ([]resources.KeyValue) {
   }
  },
  (resources.ZoneInfo) {
   Name: (string) (len=10) "us-south-3",
   DisplayName: (string) (len=10) "us-south-3",
   Status: (resources.ZoneStatus) (len=9) "Available",
   KeyValueList: ([]resources.KeyValue) {
   }
  }
 },
 KeyValueList: ([]resources.KeyValue) {
 }
}
[CLOUD-BARISTA].[INFO]: 2023-09-26 15:20:33 Test_Resources.go:1415, main.testRegionZoneHandler() - Finish GetRegionZone()
*/

/*
== ListOrgRegion() 결과 값 예시 ==
{
  "list": [
    {
      "endpoint": "https://au-syd.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/au-syd",
      "name": "au-syd",
      "status": "available"
    },
    {
      "endpoint": "https://br-sao.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/br-sao",
      "name": "br-sao",
      "status": "available"
    },
    {
      "endpoint": "https://ca-tor.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/ca-tor",
      "name": "ca-tor",
      "status": "available"
    },
    {
      "endpoint": "https://eu-de.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/eu-de",
      "name": "eu-de",
      "status": "available"
    },
    {
      "endpoint": "https://eu-es.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/eu-es",
      "name": "eu-es",
      "status": "available"
    },
    {
      "endpoint": "https://eu-gb.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/eu-gb",
      "name": "eu-gb",
      "status": "available"
    },
    {
      "endpoint": "https://jp-osa.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/jp-osa",
      "name": "jp-osa",
      "status": "available"
    },
    {
      "endpoint": "https://jp-tok.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/jp-tok",
      "name": "jp-tok",
      "status": "available"
    },
    {
      "endpoint": "https://us-east.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/us-east",
      "name": "us-east",
      "status": "available"
    },
    {
      "endpoint": "https://us-south.iaas.cloud.ibm.com",
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/us-south",
      "name": "us-south",
      "status": "available"
    }
  ]
}
*/

/*
== ListOrgZone() 결과 값 예시 ==
{
  "list": [
    {
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/us-south/zones/us-south-1",
      "name": "us-south-1",
      "region": {
        "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/us-south",
        "name": "us-south"
      },
      "status": "available"
    },
    {
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/us-south/zones/us-south-2",
      "name": "us-south-2",
      "region": {
        "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/us-south",
        "name": "us-south"
      },
      "status": "available"
    },
    {
      "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/us-south/zones/us-south-3",
      "name": "us-south-3",
      "region": {
        "href": "https://us-south.iaas.cloud.ibm.com/v1/regions/us-south",
        "name": "us-south"
      },
      "status": "available"
    }
  ]
}
*/
