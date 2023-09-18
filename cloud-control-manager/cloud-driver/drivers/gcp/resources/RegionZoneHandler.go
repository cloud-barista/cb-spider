package resources

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	compute "google.golang.org/api/compute/v1"
)

type GCPRegionZoneHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

// required Compute Engine IAM ROLE : compute.regions.list
func (regionZoneHandler *GCPRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	var regionZoneInfoList []*irs.RegionZoneInfo
	projectID := regionZoneHandler.Credential.ProjectID
	//prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
	//GET https://compute.googleapis.com/compute/v1/projects/{project}/regions

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   regionZoneHandler.Region.Zone,
		ResourceType: call.REGIONZONE,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	resp, err := regionZoneHandler.Client.Regions.List(projectID).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		cblogger.Error(err)
		return regionZoneInfoList, err
	}
	if resp == nil {
		return nil, errors.New("Not Found : Region Zone information not found")
	}

	for _, item := range resp.Items {
		info := irs.RegionZoneInfo{}
		info.Name = item.Name
		info.DisplayName = item.Name

		// ZoneList
		var zoneInfoList []*irs.ZoneInfo
		resultZones, err := GetZoneListByRegion(regionZoneHandler.Client, item.SelfLink)
		if err != nil {
			// failed to get ZoneInfo by region
		}
		for _, zone := range resultZones.Items {
			zoneInfo := irs.ZoneInfo{}
			zoneInfo.Name = zone.Name
			zoneInfo.DisplayName = zone.Name
			zoneInfo.Status = GetZoneStatus(zone.Status)

			zoneInfoList = append(zoneInfoList, &zoneInfo)
		}

		// 가져온 결과에서 Zone 정보 추출 : Zone의 status를 찾지 못해 조회하는 것으로 변경
		// for _, zoneUrl := range item.Zones {
		// 	// "https://www.googleapis.com/compute/v1/projects/csta-349809/zones/northamerica-northeast1-a"
		// 	startIndex := strings.Index(zoneUrl, "/zones/") + len("/zones/")
		// 	if startIndex < len("/zones/") {
		// 		//fmt.Println("Invalid URL:", zoneUrl)
		// 		cblogger.Error("Invalid URL:", zoneUrl)
		// 		continue
		// 	}
		// 	zone := zoneUrl[startIndex:]

		// 	zoneInfo := irs.ZoneInfo{}
		// 	zoneInfo.Name = zone
		// 	zoneInfo.DisplayName = zone

		// 	zoneInfoList = append(zoneInfoList, &zoneInfo)
		// }

		keyValueList := []irs.KeyValue{}
		itemType := reflect.TypeOf(item)
		if itemType.Kind() == reflect.Ptr {
			itemType = itemType.Elem()
		}
		itemValue := reflect.ValueOf(item)
		if itemValue.Kind() == reflect.Ptr {
			itemValue = itemValue.Elem()
		}
		numFields := itemType.NumField()

		// 속성 이름과 값을 출력합니다.
		for i := 0; i < numFields; i++ {
			field := itemType.Field(i)
			value := itemValue.Field(i).Interface()

			keyValue := irs.KeyValue{}
			keyValue.Key = field.Name
			keyValue.Value = fmt.Sprintf("%v", value)
			keyValueList = append(keyValueList, keyValue)
		}
		info.KeyValueList = keyValueList

		regionZoneInfoList = append(regionZoneInfoList, &info)
	}

	return regionZoneInfoList, nil
}
func (regionZoneHandler *GCPRegionZoneHandler) ListOrgRegion() (string, error) {

	projectID := regionZoneHandler.Credential.ProjectID

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   regionZoneHandler.Region.Zone,
		ResourceType: call.REGIONZONE,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	resp, err := regionZoneHandler.Client.Regions.List(projectID).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))
	j, _ := resp.MarshalJSON()

	//callogger.Info(j)
	return string(j), err
}
func (regionZoneHandler *GCPRegionZoneHandler) ListOrgZone() (string, error) {
	projectID := regionZoneHandler.Credential.ProjectID

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   regionZoneHandler.Region.Zone,
		ResourceType: call.REGIONZONE,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	resp, err := regionZoneHandler.Client.Zones.List(projectID).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))
	j, _ := resp.MarshalJSON()

	//callogger.Info(j)
	return string(j), err
}

/**
// Region 조회 성공 시 result
{
  "kind": "compute#regionList",
  "id": "projects/csta-349809/regions",
  "items": [
    {
      "kind": "compute#region",
      "id": "1510",
      "creationTimestamp": "1969-12-31T16:00:00.000-08:00",
      "name": "europe-west8",
      "description": "europe-west8",
      "status": "UP",
      "zones": [
        "https://www.googleapis.com/compute/v1/projects/csta-349809/zones/europe-west8-a",
        "https://www.googleapis.com/compute/v1/projects/csta-349809/zones/europe-west8-b",
        "https://www.googleapis.com/compute/v1/projects/csta-349809/zones/europe-west8-c"
      ],
      "quotas": [
		// CPUS, DISKS_TOTAL_GB ... 많아서 생략
      ],
      "selfLink": "https://www.googleapis.com/compute/v1/projects/csta-349809/regions/europe-west8",
      "supportsPzs": false
    }
  ],
  "selfLink": "https://www.googleapis.com/compute/v1/projects/csta-349809/regions"
}


// Zone 조회 성공 시
{
  "kind": "compute#zone",
  "id": "2231",
  "creationTimestamp": "1969-12-31T16:00:00.000-08:00",
  "name": "us-east1-b",
  "description": "us-east1-b",
  "status": "UP",
  "region": "https://www.googleapis.com/compute/v1/projects/csta-349809/regions/us-east1",
  "selfLink": "https://www.googleapis.com/compute/v1/projects/csta-349809/zones/us-east1-b",
  "availableCpuPlatforms": [
    "Intel Broadwell",
    "Intel Cascade Lake",
    "AMD Genoa",
    "Intel Haswell",
    "Intel Ice Lake",
    "Intel Ivy Bridge",
    "AMD Milan",
    "AMD Rome",
    "Intel Sandy Bridge",
    "Intel Sapphire Rapids",
    "Intel Skylake"
  ],
  "supportsPzs": false
}
**/
