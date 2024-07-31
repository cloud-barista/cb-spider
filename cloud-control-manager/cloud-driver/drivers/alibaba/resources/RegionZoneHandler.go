package resources

// https://next.api.alibabacloud.com/document/Ecs/2014-05-26/DescribeRegions
// https://next.api.alibabacloud.com/document/Ecs/2014-05-26/DescribeZones
// https://next.api.alibabacloud.com/api/Ecs/2014-05-26/DescribeRegions?lang=GO

import (
	"errors"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaRegionZoneHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

// 모든 Region 및 Zone 정보 조회
func (regionZoneHandler AlibabaRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	result, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		return nil, err
	}

	chanRegionZoneInfos := make(chan irs.RegionZoneInfo, len(result.Regions.Region))
	var errlist []error
	var wg sync.WaitGroup
	for _, item := range result.Regions.Region {
		wg.Add(1)
		go func(item ecs.Region) {
			defer wg.Done()
			regionId := item.RegionId
			var zoneInfoList []irs.ZoneInfo
			cblogger.Debug("regionId ", regionId)
			zonesResult, err := DescribeZonesByRegion(regionZoneHandler.Client, regionId)
			if err != nil {
				cblogger.Error("DescribeZone failed ", err.Error())
				errlist = append(errlist, err)
				return
			}
			for _, zone := range zonesResult.Zones.Zone {
				zoneInfo := irs.ZoneInfo{}
				zoneInfo.Name = zone.ZoneId
				zoneInfo.DisplayName = zone.LocalName
				zoneInfo.Status = GetZoneStatus("")
				zoneInfoList = append(zoneInfoList, zoneInfo)
			}

			info := irs.RegionZoneInfo{}
			info.Name = regionId
			info.DisplayName = item.LocalName
			info.ZoneList = zoneInfoList
			chanRegionZoneInfos <- info
		}(item)

	}
	wg.Wait()
	close(chanRegionZoneInfos)

	var regionZoneInfoList []*irs.RegionZoneInfo
	for regionZoneInfo := range chanRegionZoneInfos {
		insertRegionZoneInfo := regionZoneInfo
		regionZoneInfoList = append(regionZoneInfoList, &insertRegionZoneInfo)
	}

	if len(errlist) > 0 {
		errlistjoin := errors.Join(errlist...)
		cblogger.Error("ListRegionZone() error : ", errlistjoin)
		return regionZoneInfoList, errlistjoin
	}

	return regionZoneInfoList, nil
}

// 모든 Region 정보 조회(json return)
func (regionZoneHandler AlibabaRegionZoneHandler) ListOrgRegion() (string, error) {
	result, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		return "", err
	}

	jsonString, errJson := ConvertJsonString(result.Regions)
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}

// 모든 Zone 정보 조회(json return)
// Region에 따라 zone 정보가 달려있으므로 region 조회 후 -> zone 정보 조회
func (regionZoneHandler AlibabaRegionZoneHandler) ListOrgZone() (string, error) {
	regionsResult, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		return "", err
	}

	zoneList := make(map[string]*ecs.DescribeZonesResponse)
	for _, item := range regionsResult.Regions.Region {

		regionId := item.RegionId
		zonesResult, err := DescribeZonesByRegion(regionZoneHandler.Client, regionId)

		if err != nil {
			return "", err
		}
		zoneList[regionId] = zonesResult
	}

	jsonString, errJson := ConvertJsonString(zoneList)
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}

// 특정 Region에 대한 정보 조회.
func (regionZoneHandler AlibabaRegionZoneHandler) GetRegionZone(reqRegionId string) (irs.RegionZoneInfo, error) {
	regionInfo := irs.RegionZoneInfo{}
	result, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		return regionInfo, err
	}

	for _, item := range result.Regions.Region {
		regionId := item.RegionId

		if reqRegionId != regionId {
			continue
		}

		regionInfo.Name = regionId
		regionInfo.DisplayName = item.LocalName

		// ZoneList
		var zoneInfoList []irs.ZoneInfo
		cblogger.Debug("regionId ", regionId)
		zonesResult, err := DescribeZonesByRegion(regionZoneHandler.Client, regionId)
		if err != nil {
			cblogger.Error("DescribeZone failed ", err)
		}
		for _, zone := range zonesResult.Zones.Zone {
			zoneInfo := irs.ZoneInfo{}
			zoneInfo.Name = zone.ZoneId
			zoneInfo.DisplayName = zone.LocalName
			zoneInfo.Status = GetZoneStatus("")

			zoneInfoList = append(zoneInfoList, zoneInfo)
		}
		regionInfo.ZoneList = zoneInfoList
	}
	return regionInfo, nil
}
