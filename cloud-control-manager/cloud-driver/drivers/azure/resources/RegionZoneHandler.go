package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-11-01/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"reflect"
	"sort"
	"strings"
	"sync"
)

type AzureRegionZoneHandler struct {
	CredentialInfo     idrv.CredentialInfo
	Region             idrv.RegionInfo
	Ctx                context.Context
	Client             *subscriptions.Client
	GroupsClient       *resources.GroupsClient
	ResourceSkusClient *compute.ResourceSkusClient
}

func removeDuplicateStr(array []string) []string {
	if len(array) < 1 {
		return array
	}

	sort.Strings(array)
	prev := 1
	for curr := 1; curr < len(array); curr++ {
		if array[curr-1] != array[curr] {
			array[prev] = array[curr]
			prev++
		}
	}

	return array[:prev]
}

func (regionZoneHandler *AzureRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.Region, call.REGIONZONE, "RegionZone", "ListRegionZone()")
	start := call.Start()

	resultListLocations, err := regionZoneHandler.Client.ListLocations(regionZoneHandler.Ctx,
		regionZoneHandler.CredentialInfo.SubscriptionId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	var regionZoneInfo []*irs.RegionZoneInfo

	var routineMax = 50
	var wait sync.WaitGroup
	var mutex = &sync.Mutex{}
	var lenLocations = len(*resultListLocations.Value)
	var zoneErrorOccurred bool

	for i := 0; i < lenLocations; {
		if lenLocations-i < routineMax {
			routineMax = lenLocations - i
		}

		wait.Add(routineMax)

		for j := 0; j < routineMax; j++ {
			go func(wait *sync.WaitGroup, loc subscriptions.Location) {
				var zones []string
				var zoneList []irs.ZoneInfo
				var resultResourceSkusClient compute.ResourceSkusResultPage

				resultResourceSkusClient, err = regionZoneHandler.ResourceSkusClient.List(regionZoneHandler.Ctx, "location eq '"+*loc.Name+"'")
				if err != nil {
					zoneErrorOccurred = true
					return
				}

				for _, val := range resultResourceSkusClient.Values() {
					for _, locInfo := range *val.LocationInfo {
						locName := strings.ToLower(*loc.Name)
						locloc := strings.ToLower(*locInfo.Location)

						if locName == locloc && locInfo.Zones != nil {
							for _, zone := range *locInfo.Zones {
								zones = append(zones, zone)
							}
							break
						}
					}
				}

				zones = removeDuplicateStr(zones)

				for _, zone := range zones {
					zoneList = append(zoneList, irs.ZoneInfo{
						Name:         zone,
						DisplayName:  zone,
						Status:       irs.NotSupported,
						KeyValueList: []irs.KeyValue{},
					})
				}

				mutex.Lock()
				regionZoneInfo = append(regionZoneInfo, &irs.RegionZoneInfo{
					Name:         *loc.Name,
					DisplayName:  *loc.DisplayName,
					ZoneList:     zoneList,
					KeyValueList: []irs.KeyValue{},
				})
				mutex.Unlock()

				wait.Done()
			}(&wait, (*resultListLocations.Value)[i])

			i++
			if i == lenLocations {
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

func (regionZoneHandler *AzureRegionZoneHandler) GetRegionZone(Name string) (irs.RegionZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.Region, call.REGIONZONE, "RegionZone", "GetRegionZone()")
	start := call.Start()

	resultListLocations, err := regionZoneHandler.Client.ListLocations(regionZoneHandler.Ctx,
		regionZoneHandler.CredentialInfo.SubscriptionId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.RegionZoneInfo{}, getErr
	}

	resultResourceSkusClient, err := regionZoneHandler.ResourceSkusClient.List(regionZoneHandler.Ctx, "location eq '"+Name+"'")
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get RegionZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.RegionZoneInfo{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	var location *subscriptions.Location
	var regionZoneInfo irs.RegionZoneInfo

	for _, loc := range *resultListLocations.Value {
		if *loc.Name == Name {
			location = &loc
			break
		}
	}

	regionZoneInfo.Name = *location.Name
	regionZoneInfo.DisplayName = *location.DisplayName

	var zones []string

	for _, val := range resultResourceSkusClient.Values() {
		for _, locInfo := range *val.LocationInfo {
			locName := strings.ToLower(Name)
			locloc := strings.ToLower(*locInfo.Location)

			if locName == locloc && locInfo.Zones != nil {
				for _, zone := range *locInfo.Zones {
					zones = append(zones, zone)
				}
				break
			}
		}
	}

	zones = removeDuplicateStr(zones)

	for _, zone := range zones {
		regionZoneInfo.ZoneList = append(regionZoneInfo.ZoneList, irs.ZoneInfo{
			Name:         zone,
			DisplayName:  zone,
			Status:       irs.NotSupported,
			KeyValueList: []irs.KeyValue{},
		})
	}

	elements := reflect.ValueOf(location.Metadata).Elem()
	for index := 0; index < elements.NumField(); index++ {
		var value any

		if elements.Field(index).Kind() == reflect.Struct {
			continue
		} else if elements.Field(index).Kind() == reflect.Pointer && !elements.Field(index).IsNil() {
			if elements.Field(index).Elem().Kind() == reflect.Struct || elements.Field(index).Elem().Kind() == reflect.Slice {
				continue
			}
			value = elements.Field(index).Elem().Interface()
		} else {
			value = elements.Field(index)
		}

		typeField := elements.Type().Field(index)
		regionZoneInfo.KeyValueList = append(regionZoneInfo.KeyValueList, irs.KeyValue{
			Key:   typeField.Name,
			Value: fmt.Sprintf("%+v", value),
		})
	}

	return regionZoneInfo, nil
}

func (regionZoneHandler *AzureRegionZoneHandler) ListOrgRegion() (string, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.Region, call.REGIONZONE, "RegionZone", "ListOrgRegion()")
	start := call.Start()
	result, err := regionZoneHandler.Client.ListLocations(regionZoneHandler.Ctx,
		regionZoneHandler.CredentialInfo.SubscriptionId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgRegion. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	LoggingInfo(hiscallInfo, start)

	type location struct {
		// ID - READ-ONLY; The fully qualified ID of the location. For example, /subscriptions/00000000-0000-0000-0000-000000000000/locations/westus.
		ID string `json:"id,omitempty"`
		// SubscriptionID - READ-ONLY; The subscription ID.
		SubscriptionID string `json:"subscriptionId,omitempty"`
		// Name - READ-ONLY; The location name.
		Name string `json:"name,omitempty"`
		// DisplayName - READ-ONLY; The display name of the location.
		DisplayName string `json:"displayName,omitempty"`
		// RegionalDisplayName - READ-ONLY; The display name of the location and its region.
		RegionalDisplayName string `json:"regionalDisplayName,omitempty"`
	}

	var locationList struct {
		List []location `json:"list"`
	}

	for _, loc := range *result.Value {
		locationList.List = append(locationList.List, location{
			ID:                  *loc.ID,
			Name:                *loc.Name,
			DisplayName:         *loc.DisplayName,
			RegionalDisplayName: *loc.RegionalDisplayName,
		})
	}

	jsonBytes, err := json.Marshal(locationList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgRegion. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

func (regionZoneHandler *AzureRegionZoneHandler) ListOrgZone() (string, error) {
	hiscallInfo := GetCallLogScheme(regionZoneHandler.Region, call.REGIONZONE, "RegionZone", "ListOrgZone()")
	start := call.Start()

	resultGroupsClient, err := regionZoneHandler.GroupsClient.Get(regionZoneHandler.Ctx, regionZoneHandler.Region.ResourceGroup)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	resultResourceSkusClient, err := regionZoneHandler.ResourceSkusClient.List(regionZoneHandler.Ctx, "")
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgZone. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	LoggingInfo(hiscallInfo, start)

	var result struct {
		Zones []string `json:"zones"`
	}

	for _, val := range resultResourceSkusClient.Values() {
		for _, locInfo := range *val.LocationInfo {
			loc := strings.ToLower(*locInfo.Location)
			region := strings.ToLower(*resultGroupsClient.Location)

			if loc == region && locInfo.Zones != nil {
				for _, zone := range *locInfo.Zones {
					result.Zones = append(result.Zones, zone)
				}
				break
			}
		}
	}

	result.Zones = removeDuplicateStr(result.Zones)

	jsonBytes, err := json.Marshal(result)
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
[CLOUD-BARISTA].[INFO]: 2023-09-22 17:44:24 Test_Resources.go:1273, main.testRegionZoneHandler() - Start GetRegionZone() ...
Enter Region Name: koreacentral
[CLOUD-BARISTA].[INFO]: 2023-09-22 17:44:28 CommonAzureFunc.go:52, github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/resources.GetCallLogScheme() - Call AZURE GetRegionZone()
[HISCALL].[124.53.55.55] 2023-09-22 17:44:29 (Friday) github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/resources.LoggingInfo():48 - "CloudOS" : "AZURE", "RegionZone" : "Korea Central", "ResourceType" : "REGIONZONE", "ResourceName" : "RegionZone", "CloudOSAPI" : "GetRegionZone()", "ElapsedTime" : "1.9558", "ErrorMSG" : ""
(resources.RegionZoneInfo) {
 Name: (string) (len=12) "koreacentral",
 DisplayName: (string) (len=13) "Korea Central",
 ZoneList: ([]resources.ZoneInfo) (len=3 cap=4) {
  (resources.ZoneInfo) {
   Name: (string) (len=1) "1",
   DisplayName: (string) (len=1) "1",
   Status: (resources.ZoneStatus) (len=12) "NotSupported",
   KeyValueList: ([]resources.KeyValue) {
   }
  },
  (resources.ZoneInfo) {
   Name: (string) (len=1) "2",
   DisplayName: (string) (len=1) "2",
   Status: (resources.ZoneStatus) (len=12) "NotSupported",
   KeyValueList: ([]resources.KeyValue) {
   }
  },
  (resources.ZoneInfo) {
   Name: (string) (len=1) "3",
   DisplayName: (string) (len=1) "3",
   Status: (resources.ZoneStatus) (len=12) "NotSupported",
   KeyValueList: ([]resources.KeyValue) {
   }
  }
 },
 KeyValueList: ([]resources.KeyValue) {
 }
}
[CLOUD-BARISTA].[INFO]: 2023-09-22 17:44:29 Test_Resources.go:1284, main.testRegionZoneHandler() - Finish GetRegionZone()
*/

/*
== ListOrgRegion() 결과 값 예시 ==
{
  "list": [
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/eastus",
      "name": "eastus",
      "displayName": "East US",
      "regionalDisplayName": "(US) East US"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/eastus2",
      "name": "eastus2",
      "displayName": "East US 2",
      "regionalDisplayName": "(US) East US 2"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southcentralus",
      "name": "southcentralus",
      "displayName": "South Central US",
      "regionalDisplayName": "(US) South Central US"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/westus2",
      "name": "westus2",
      "displayName": "West US 2",
      "regionalDisplayName": "(US) West US 2"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/westus3",
      "name": "westus3",
      "displayName": "West US 3",
      "regionalDisplayName": "(US) West US 3"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/australiaeast",
      "name": "australiaeast",
      "displayName": "Australia East",
      "regionalDisplayName": "(Asia Pacific) Australia East"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southeastasia",
      "name": "southeastasia",
      "displayName": "Southeast Asia",
      "regionalDisplayName": "(Asia Pacific) Southeast Asia"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/northeurope",
      "name": "northeurope",
      "displayName": "North Europe",
      "regionalDisplayName": "(Europe) North Europe"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/swedencentral",
      "name": "swedencentral",
      "displayName": "Sweden Central",
      "regionalDisplayName": "(Europe) Sweden Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/uksouth",
      "name": "uksouth",
      "displayName": "UK South",
      "regionalDisplayName": "(Europe) UK South"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/westeurope",
      "name": "westeurope",
      "displayName": "West Europe",
      "regionalDisplayName": "(Europe) West Europe"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/centralus",
      "name": "centralus",
      "displayName": "Central US",
      "regionalDisplayName": "(US) Central US"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southafricanorth",
      "name": "southafricanorth",
      "displayName": "South Africa North",
      "regionalDisplayName": "(Africa) South Africa North"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/centralindia",
      "name": "centralindia",
      "displayName": "Central India",
      "regionalDisplayName": "(Asia Pacific) Central India"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/eastasia",
      "name": "eastasia",
      "displayName": "East Asia",
      "regionalDisplayName": "(Asia Pacific) East Asia"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/japaneast",
      "name": "japaneast",
      "displayName": "Japan East",
      "regionalDisplayName": "(Asia Pacific) Japan East"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/koreacentral",
      "name": "koreacentral",
      "displayName": "Korea Central",
      "regionalDisplayName": "(Asia Pacific) Korea Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/canadacentral",
      "name": "canadacentral",
      "displayName": "Canada Central",
      "regionalDisplayName": "(Canada) Canada Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/francecentral",
      "name": "francecentral",
      "displayName": "France Central",
      "regionalDisplayName": "(Europe) France Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/germanywestcentral",
      "name": "germanywestcentral",
      "displayName": "Germany West Central",
      "regionalDisplayName": "(Europe) Germany West Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/italynorth",
      "name": "italynorth",
      "displayName": "Italy North",
      "regionalDisplayName": "(Europe) Italy North"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/norwayeast",
      "name": "norwayeast",
      "displayName": "Norway East",
      "regionalDisplayName": "(Europe) Norway East"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/polandcentral",
      "name": "polandcentral",
      "displayName": "Poland Central",
      "regionalDisplayName": "(Europe) Poland Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/switzerlandnorth",
      "name": "switzerlandnorth",
      "displayName": "Switzerland North",
      "regionalDisplayName": "(Europe) Switzerland North"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/uaenorth",
      "name": "uaenorth",
      "displayName": "UAE North",
      "regionalDisplayName": "(Middle East) UAE North"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/brazilsouth",
      "name": "brazilsouth",
      "displayName": "Brazil South",
      "regionalDisplayName": "(South America) Brazil South"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/centraluseuap",
      "name": "centraluseuap",
      "displayName": "Central US EUAP",
      "regionalDisplayName": "(US) Central US EUAP"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/qatarcentral",
      "name": "qatarcentral",
      "displayName": "Qatar Central",
      "regionalDisplayName": "(Middle East) Qatar Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/centralusstage",
      "name": "centralusstage",
      "displayName": "Central US (Stage)",
      "regionalDisplayName": "(US) Central US (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/eastusstage",
      "name": "eastusstage",
      "displayName": "East US (Stage)",
      "regionalDisplayName": "(US) East US (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/eastus2stage",
      "name": "eastus2stage",
      "displayName": "East US 2 (Stage)",
      "regionalDisplayName": "(US) East US 2 (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/northcentralusstage",
      "name": "northcentralusstage",
      "displayName": "North Central US (Stage)",
      "regionalDisplayName": "(US) North Central US (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southcentralusstage",
      "name": "southcentralusstage",
      "displayName": "South Central US (Stage)",
      "regionalDisplayName": "(US) South Central US (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/westusstage",
      "name": "westusstage",
      "displayName": "West US (Stage)",
      "regionalDisplayName": "(US) West US (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/westus2stage",
      "name": "westus2stage",
      "displayName": "West US 2 (Stage)",
      "regionalDisplayName": "(US) West US 2 (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/asia",
      "name": "asia",
      "displayName": "Asia",
      "regionalDisplayName": "Asia"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/asiapacific",
      "name": "asiapacific",
      "displayName": "Asia Pacific",
      "regionalDisplayName": "Asia Pacific"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/australia",
      "name": "australia",
      "displayName": "Australia",
      "regionalDisplayName": "Australia"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/brazil",
      "name": "brazil",
      "displayName": "Brazil",
      "regionalDisplayName": "Brazil"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/canada",
      "name": "canada",
      "displayName": "Canada",
      "regionalDisplayName": "Canada"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/europe",
      "name": "europe",
      "displayName": "Europe",
      "regionalDisplayName": "Europe"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/france",
      "name": "france",
      "displayName": "France",
      "regionalDisplayName": "France"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/germany",
      "name": "germany",
      "displayName": "Germany",
      "regionalDisplayName": "Germany"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/global",
      "name": "global",
      "displayName": "Global",
      "regionalDisplayName": "Global"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/india",
      "name": "india",
      "displayName": "India",
      "regionalDisplayName": "India"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/japan",
      "name": "japan",
      "displayName": "Japan",
      "regionalDisplayName": "Japan"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/korea",
      "name": "korea",
      "displayName": "Korea",
      "regionalDisplayName": "Korea"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/norway",
      "name": "norway",
      "displayName": "Norway",
      "regionalDisplayName": "Norway"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/singapore",
      "name": "singapore",
      "displayName": "Singapore",
      "regionalDisplayName": "Singapore"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southafrica",
      "name": "southafrica",
      "displayName": "South Africa",
      "regionalDisplayName": "South Africa"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/switzerland",
      "name": "switzerland",
      "displayName": "Switzerland",
      "regionalDisplayName": "Switzerland"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/uae",
      "name": "uae",
      "displayName": "United Arab Emirates",
      "regionalDisplayName": "United Arab Emirates"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/uk",
      "name": "uk",
      "displayName": "United Kingdom",
      "regionalDisplayName": "United Kingdom"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/unitedstates",
      "name": "unitedstates",
      "displayName": "United States",
      "regionalDisplayName": "United States"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/unitedstateseuap",
      "name": "unitedstateseuap",
      "displayName": "United States EUAP",
      "regionalDisplayName": "United States EUAP"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/eastasiastage",
      "name": "eastasiastage",
      "displayName": "East Asia (Stage)",
      "regionalDisplayName": "(Asia Pacific) East Asia (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southeastasiastage",
      "name": "southeastasiastage",
      "displayName": "Southeast Asia (Stage)",
      "regionalDisplayName": "(Asia Pacific) Southeast Asia (Stage)"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/brazilus",
      "name": "brazilus",
      "displayName": "Brazil US",
      "regionalDisplayName": "(South America) Brazil US"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/eastusstg",
      "name": "eastusstg",
      "displayName": "East US STG",
      "regionalDisplayName": "(US) East US STG"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/northcentralus",
      "name": "northcentralus",
      "displayName": "North Central US",
      "regionalDisplayName": "(US) North Central US"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/westus",
      "name": "westus",
      "displayName": "West US",
      "regionalDisplayName": "(US) West US"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/jioindiawest",
      "name": "jioindiawest",
      "displayName": "Jio India West",
      "regionalDisplayName": "(Asia Pacific) Jio India West"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/eastus2euap",
      "name": "eastus2euap",
      "displayName": "East US 2 EUAP",
      "regionalDisplayName": "(US) East US 2 EUAP"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southcentralusstg",
      "name": "southcentralusstg",
      "displayName": "South Central US STG",
      "regionalDisplayName": "(US) South Central US STG"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/westcentralus",
      "name": "westcentralus",
      "displayName": "West Central US",
      "regionalDisplayName": "(US) West Central US"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southafricawest",
      "name": "southafricawest",
      "displayName": "South Africa West",
      "regionalDisplayName": "(Africa) South Africa West"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/australiacentral",
      "name": "australiacentral",
      "displayName": "Australia Central",
      "regionalDisplayName": "(Asia Pacific) Australia Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/australiacentral2",
      "name": "australiacentral2",
      "displayName": "Australia Central 2",
      "regionalDisplayName": "(Asia Pacific) Australia Central 2"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/australiasoutheast",
      "name": "australiasoutheast",
      "displayName": "Australia Southeast",
      "regionalDisplayName": "(Asia Pacific) Australia Southeast"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/japanwest",
      "name": "japanwest",
      "displayName": "Japan West",
      "regionalDisplayName": "(Asia Pacific) Japan West"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/jioindiacentral",
      "name": "jioindiacentral",
      "displayName": "Jio India Central",
      "regionalDisplayName": "(Asia Pacific) Jio India Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/koreasouth",
      "name": "koreasouth",
      "displayName": "Korea South",
      "regionalDisplayName": "(Asia Pacific) Korea South"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/southindia",
      "name": "southindia",
      "displayName": "South India",
      "regionalDisplayName": "(Asia Pacific) South India"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/westindia",
      "name": "westindia",
      "displayName": "West India",
      "regionalDisplayName": "(Asia Pacific) West India"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/canadaeast",
      "name": "canadaeast",
      "displayName": "Canada East",
      "regionalDisplayName": "(Canada) Canada East"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/francesouth",
      "name": "francesouth",
      "displayName": "France South",
      "regionalDisplayName": "(Europe) France South"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/germanynorth",
      "name": "germanynorth",
      "displayName": "Germany North",
      "regionalDisplayName": "(Europe) Germany North"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/norwaywest",
      "name": "norwaywest",
      "displayName": "Norway West",
      "regionalDisplayName": "(Europe) Norway West"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/switzerlandwest",
      "name": "switzerlandwest",
      "displayName": "Switzerland West",
      "regionalDisplayName": "(Europe) Switzerland West"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/ukwest",
      "name": "ukwest",
      "displayName": "UK West",
      "regionalDisplayName": "(Europe) UK West"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/uaecentral",
      "name": "uaecentral",
      "displayName": "UAE Central",
      "regionalDisplayName": "(Middle East) UAE Central"
    },
    {
      "id": "/subscriptions/cf02e63f-eb85-49c3-9ca3-303bb808d9fb/locations/brazilsoutheast",
      "name": "brazilsoutheast",
      "displayName": "Brazil Southeast",
      "regionalDisplayName": "(South America) Brazil Southeast"
    }
  ]
}
*/

/*
== ListOrgZone() 결과 값 예시 ==
{
  "zones": [
    "1",
    "2",
    "3"
  ]
}
*/
