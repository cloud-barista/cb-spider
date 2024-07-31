package resources

import (
	"errors"
	"sync"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentRegionZoneHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

func (regionZoneHandler *TencentRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {

	responseRegions, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		cblogger.Error("err : DescribeRegions")
		cblogger.Error(err)
		return nil, err
	}

	chanRegionZoneInfos := make(chan irs.RegionZoneInfo, len(responseRegions.Response.RegionSet))
	var wg sync.WaitGroup
	var errlist []error
	clientProfile := profile.NewClientProfile()
	clientProfile.Language = "en-US" // lang default set is zh-CN -> set as en-US.

	for _, region := range responseRegions.Response.RegionSet {
		wg.Add(1)
		go func(region *cvm.RegionInfo) {
			defer wg.Done()
			tempClient, err := cvm.NewClient(regionZoneHandler.Client.Client.GetCredential(), *region.Region, clientProfile)
			if err != nil {
				cblogger.Error("NewClient failed on ", region, err.Error())
				errlist = append(errlist, err)
				return
			}
			responseZones, err := DescribeZones(tempClient)
			if err != nil {
				cblogger.Error("DescribeZones failed ", err.Error())
				errlist = append(errlist, err)
				return
			}
			var zoneInfoList []irs.ZoneInfo
			for _, zone := range responseZones.Response.ZoneSet {
				zoneInfo := irs.ZoneInfo{}
				zoneInfo.Name = *zone.Zone
				zoneInfo.DisplayName = *zone.ZoneName
				zoneInfo.Status = GetZoneStatus(*zone.ZoneState)
				zoneInfoList = append(zoneInfoList, zoneInfo)
			}

			regionInfo := irs.RegionZoneInfo{}
			regionInfo.Name = *region.Region
			regionInfo.DisplayName = *region.RegionName
			regionInfo.ZoneList = zoneInfoList
			chanRegionZoneInfos <- regionInfo

		}(region)

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

func (regionZoneHandler *TencentRegionZoneHandler) GetRegionZone(Name string) (irs.RegionZoneInfo, error) {
	responseRegions, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		cblogger.Error("err : DescribeRegions")
		cblogger.Error(err)
		return irs.RegionZoneInfo{}, err
	}

	var targetRegion cvm.RegionInfo
	for _, region := range responseRegions.Response.RegionSet {
		if *region.Region == Name {
			targetRegion = *region
		}
	}

	clientProfile := profile.NewClientProfile()
	clientProfile.Language = "en-US" // lang default set is zh-CN -> set as en-US.

	// var regionZoneInfo irs.RegionZoneInfo

	tempClient, _ := cvm.NewClient(regionZoneHandler.Client.Client.GetCredential(), Name, clientProfile)
	responseZones, _ := DescribeZones(tempClient)

	var zoneInfoList []irs.ZoneInfo
	for _, zone := range responseZones.Response.ZoneSet {
		zoneInfo := irs.ZoneInfo{}
		zoneInfo.Name = *zone.Zone
		zoneInfo.DisplayName = *zone.ZoneName
		zoneInfo.Status = GetZoneStatus(*zone.ZoneState)

		// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
		// keyValueList, err := ConvertKeyValueList(zone)
		// if err != nil {
		// 	cblogger.Errorf("err : ConvertKeyValueList [%s]", *zone.ZoneName)
		// 	cblogger.Error(err)
		// 	keyValueList = nil
		// }
		// zoneInfo.KeyValueList = keyValueList

		zoneInfoList = append(zoneInfoList, zoneInfo)
	}

	regionZoneInfo := irs.RegionZoneInfo{}
	regionZoneInfo.Name = *targetRegion.Region
	regionZoneInfo.DisplayName = *targetRegion.RegionName
	regionZoneInfo.ZoneList = zoneInfoList

	// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
	// keyValueList, err := ConvertKeyValueList(targetRegion)
	// if err != nil {
	// 	cblogger.Errorf("err : ConvertKeyValueList [%s]", Name)
	// 	cblogger.Error(err)
	// 	keyValueList = nil
	// }
	// regionZoneInfo.KeyValueList = keyValueList

	return regionZoneInfo, nil
}

func (regionZoneHandler *TencentRegionZoneHandler) ListOrgRegion() (string, error) {

	responseRegions, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	jsonString, errJson := ConvertJsonString(responseRegions)
	if errJson != nil {
		cblogger.Error(err)
		return "", err
	}

	return jsonString, err
}

func (regionZoneHandler *TencentRegionZoneHandler) ListOrgZone() (string, error) {

	responseRegions, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	clientProfile := profile.NewClientProfile()
	clientProfile.Language = "en-US" // lang default set is zh-CN -> set as en-US.

	var responseZonesList []*cvm.DescribeZonesResponse

	for _, region := range responseRegions.Response.RegionSet {
		tempClient, _ := cvm.NewClient(regionZoneHandler.Client.Client.GetCredential(), *region.Region, clientProfile)
		responseZones, _ := DescribeZones(tempClient)

		responseZonesList = append(responseZonesList, responseZones)
	}

	jsonString, errJson := ConvertJsonString(responseZonesList)
	if errJson != nil {
		cblogger.Error(err)
		return "", err
	}

	return jsonString, nil
}
