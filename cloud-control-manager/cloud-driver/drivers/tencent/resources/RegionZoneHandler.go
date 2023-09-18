package resources

import (
	"reflect"

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

	clientProfile := profile.NewClientProfile()
	clientProfile.Language = "en-US" // lang default set is zh-CN -> set as en-US.

	var regionZoneInfoList []*irs.RegionZoneInfo
	for _, region := range responseRegions.Response.RegionSet {
		tempClient, _ := cvm.NewClient(regionZoneHandler.Client.Client.GetCredential(), *region.Region, clientProfile)
		responseZones, _ := DescribeZones(tempClient)

		var zoneInfoList []irs.ZoneInfo
		for _, zone := range responseZones.Response.ZoneSet {
			keyValueList := []irs.KeyValue{}
			itemType := reflect.TypeOf(zone)
			if itemType.Kind() == reflect.Ptr {
				itemType = itemType.Elem()
			}
			itemValue := reflect.ValueOf(zone)
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
				keyValue.Value = *value.(*string)

				keyValueList = append(keyValueList, keyValue)
			}

			zoneInfo := irs.ZoneInfo{}
			zoneInfo.Name = *zone.Zone
			zoneInfo.DisplayName = *zone.ZoneName
			zoneInfo.Status = GetZoneStatus(*zone.ZoneState)
			zoneInfo.KeyValueList = keyValueList

			zoneInfoList = append(zoneInfoList, zoneInfo)
		}

		keyValueList := []irs.KeyValue{}
		itemType := reflect.TypeOf(region)
		if itemType.Kind() == reflect.Ptr {
			itemType = itemType.Elem()
		}
		itemValue := reflect.ValueOf(region)
		if itemValue.Kind() == reflect.Ptr {
			itemValue = itemValue.Elem()
		}
		numFields := itemType.NumField()

		for i := 0; i < numFields; i++ {
			field := itemType.Field(i)
			value := itemValue.Field(i).Interface()

			keyValue := irs.KeyValue{}
			keyValue.Key = field.Name
			keyValue.Value = *value.(*string)

			keyValueList = append(keyValueList, keyValue)
		}

		regionInfo := irs.RegionZoneInfo{}
		regionInfo.Name = *region.Region
		regionInfo.DisplayName = *region.RegionName
		regionInfo.ZoneList = zoneInfoList
		regionInfo.KeyValueList = keyValueList

		regionZoneInfoList = append(regionZoneInfoList, &regionInfo)
	}

	return regionZoneInfoList, nil
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
