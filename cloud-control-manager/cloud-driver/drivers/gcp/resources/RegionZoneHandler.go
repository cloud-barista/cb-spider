package resources

import (
	"context"
	"errors"
	"sync"

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

// GetRegionZone implements resources.RegionZoneHandler.
// 특정 region 정보만 가져올 때.  regions.list에 filter 조건으로 name=asia-east1 을 추가해도 되나 get api가 있어 해당 api 사용
func (regionZoneHandler *GCPRegionZoneHandler) GetRegionZone(regionName string) (irs.RegionZoneInfo, error) {
	var regionZoneInfo irs.RegionZoneInfo
	projectID := regionZoneHandler.Credential.ProjectID

	resp, err := GetRegion(regionZoneHandler.Client, projectID, regionName)
	if err != nil {
		cblogger.Error(err)
		return regionZoneInfo, err
	}
	regionZoneInfo.Name = resp.Name
	regionZoneInfo.DisplayName = resp.Name

	// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
	// regionZoneInfo.KeyValueList, err = ConvertKeyValueList(resp)
	// if err != nil {
	// 	regionZoneInfo.KeyValueList = nil
	// 	cblogger.Error(err)
	// }

	// ZoneList
	var zoneInfoList []irs.ZoneInfo
	resultZones, err := GetZoneListByRegion(regionZoneHandler.Client, projectID, resp.SelfLink)
	if err != nil {
		// failed to get ZoneInfo by region
		cblogger.Error(err)
	} else {
		if resultZones != nil && resultZones.Items != nil {
			for _, zone := range resultZones.Items {
				zoneInfo := irs.ZoneInfo{}
				zoneInfo.Name = zone.Name
				zoneInfo.DisplayName = zone.Name
				zoneInfo.Status = GetZoneStatus(zone.Status)

				// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
				// zoneInfo.KeyValueList, err = ConvertKeyValueList(zone)
				// if err != nil {
				// 	zoneInfo.KeyValueList = nil
				// 	cblogger.Error(err)
				// }

				zoneInfoList = append(zoneInfoList, zoneInfo)
				// set zone keyvalue list
			}
			regionZoneInfo.ZoneList = zoneInfoList
		}
	}

	// set region keyvalue list

	return regionZoneInfo, nil
}

// required Compute Engine IAM ROLE : compute.regions.list
func (regionZoneHandler *GCPRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	projectID := regionZoneHandler.Credential.ProjectID
	resp, err := ListRegion(regionZoneHandler.Client, projectID)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("Not Found : Region Zone information not found")
	}

	chanRegionZoneInfos := make(chan irs.RegionZoneInfo, len(resp.Items))
	var wg sync.WaitGroup

	var errlist []error

	for _, item := range resp.Items {
		wg.Add(1)
		go func(item *compute.Region) {
			defer wg.Done()

			// ZoneList
			var zoneInfoList []irs.ZoneInfo
			resultZones, err := GetZoneListByRegion(regionZoneHandler.Client, projectID, item.SelfLink)
			if err != nil {
				// failed to get ZoneInfo by region
				cblogger.Error("DescribeZone failed ", err.Error())
				errlist = append(errlist, err)
				return
			}
			for _, zone := range resultZones.Items {

				zoneInfo := irs.ZoneInfo{}
				zoneInfo.Name = zone.Name
				zoneInfo.DisplayName = zone.Name
				zoneInfo.Status = GetZoneStatus(zone.Status)

				// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
				// zoneInfo.KeyValueList, err = ConvertKeyValueList(zone)
				// if err != nil {
				// 	zoneInfo.KeyValueList = nil
				// 	cblogger.Error(err)
				// }

				zoneInfoList = append(zoneInfoList, zoneInfo)
			}

			// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
			// info.KeyValueList, err = ConvertKeyValueList(item)
			// if err != nil {
			// 	info.KeyValueList = nil
			// 	cblogger.Error(err)
			// }
			info := irs.RegionZoneInfo{}
			info.Name = item.Name
			info.DisplayName = item.Name
			info.ZoneList = zoneInfoList
			chanRegionZoneInfos <- info
			// regionZoneInfoList = append(regionZoneInfoList, &info)
		}(item)
	}
	// set keyvalue list

	wg.Wait()
	close(chanRegionZoneInfos)

	var regionZoneInfoList []*irs.RegionZoneInfo
	for regionZoneInfo := range chanRegionZoneInfos {
		insertRegionZoneInfo := regionZoneInfo
		regionZoneInfoList = append(regionZoneInfoList, &insertRegionZoneInfo)
	}

	if len(errlist) > 0 {
		errlistjoin := errors.Join(errlist...)
		return regionZoneInfoList, errlistjoin
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
