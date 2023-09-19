package resources

// https://next.api.alibabacloud.com/document/Ecs/2014-05-26/DescribeRegions
// https://next.api.alibabacloud.com/document/Ecs/2014-05-26/DescribeZones
// https://next.api.alibabacloud.com/api/Ecs/2014-05-26/DescribeRegions?lang=GO

import (
	"fmt"
	"reflect"

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
	var regionZoneInfoList []*irs.RegionZoneInfo

	// request := ecs.CreateDescribeRegionsRequest()
	// request.AcceptLanguage = "en-US" // Only Chinese (zh-CN : default), English (en-US), and Japanese (ja) are allowed

	// callogger := call.GetLogger("HISCALL")
	// callLogInfo := call.CLOUDLOGSCHEMA{
	// 	CloudOS:      call.ALIBABA,
	// 	RegionZone:   regionZoneHandler.Region.Zone,
	// 	ResourceType: call.REGIONZONE,
	// 	ResourceName: "Regions",
	// 	CloudOSAPI:   "ListRegionZone()",
	// 	ElapsedTime:  "",
	// 	ErrorMSG:     "",
	// }
	// callLogStart := call.Start()
	// result, err := regionZoneHandler.Client.DescribeRegions(request)
	// callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	// if err != nil {
	// 	callLogInfo.ErrorMSG = err.Error()
	// 	callogger.Error(call.String(callLogInfo))
	// 	return regionZoneInfoList, err
	// }
	// callogger.Info(call.String(callLogInfo))

	result, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		return regionZoneInfoList, err
	}

	for _, item := range result.Regions.Region {
		regionId := item.RegionId

		info := irs.RegionZoneInfo{}
		info.Name = regionId
		info.DisplayName = item.LocalName

		// regionStatus := GetRegionStatus(item.Status)
		// cblogger.Info("regionStatus ", regionStatus)

		// ZoneList
		var zoneInfoList []irs.ZoneInfo
		cblogger.Info("regionId ", regionId)
		zonesResult, err := DescribeZonesByRegion(regionZoneHandler.Client, regionId)
		if err != nil {
			cblogger.Debug("DescribeZone failed ", err)
		}
		for _, zone := range zonesResult.Zones.Zone {
			zoneInfo := irs.ZoneInfo{}
			zoneInfo.Name = zone.ZoneId
			zoneInfo.DisplayName = zone.LocalName
			zoneInfo.Status = GetZoneStatus("") // Zone의 상태값이 없으므로 set하지 않도록 변경.

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
				keyValue.Value = fmt.Sprintf("%v", value)
				keyValueList = append(keyValueList, keyValue)
			}
			zoneInfo.KeyValueList = keyValueList

			zoneInfoList = append(zoneInfoList, zoneInfo)
		}
		info.ZoneList = zoneInfoList
		// "ZoneType": "AvailabilityZone",
		// "LocalName": "曼谷 可用区A",
		// "ZoneId": "ap-southeast-7a",

		keyValueList := []irs.KeyValue{}
		keyValue := irs.KeyValue{}
		keyValue.Key = "RegionEndpoint"
		keyValue.Value = item.RegionEndpoint
		info.KeyValueList = keyValueList

		regionZoneInfoList = append(regionZoneInfoList, &info)
	}

	return regionZoneInfoList, err

}

// 모든 Region 정보 조회(json return)
func (regionZoneHandler AlibabaRegionZoneHandler) ListOrgRegion() (string, error) {
	// request := ecs.CreateDescribeRegionsRequest()
	// request.AcceptLanguage = "en-US" // Only Chinese (zh-CN : default), English (en-US), and Japanese (ja) are allowed

	// callogger := call.GetLogger("HISCALL")
	// callLogInfo := call.CLOUDLOGSCHEMA{
	// 	CloudOS:      call.ALIBABA,
	// 	RegionZone:   regionZoneHandler.Region.Zone,
	// 	ResourceType: call.REGIONZONE,
	// 	ResourceName: "",
	// 	CloudOSAPI:   "ListOrgRegion()",
	// 	ElapsedTime:  "",
	// 	ErrorMSG:     "",
	// }

	// callLogStart := call.Start()
	// result, err := regionZoneHandler.Client.DescribeRegions(request)
	// callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	// if err != nil {
	// 	callLogInfo.ErrorMSG = err.Error()
	// 	callogger.Error(call.String(callLogInfo))
	// 	return "", err
	// }
	// callogger.Info(call.String(callLogInfo))

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

	// request := ecs.CreateDescribeZonesRequest()
	// request.AcceptLanguage = "en-US" // Only Chinese (zh-CN : default), English (en-US), and Japanese (ja) are allowed

	// callogger := call.GetLogger("HISCALL")
	// callLogInfo := call.CLOUDLOGSCHEMA{
	// 	CloudOS:      call.ALIBABA,
	// 	RegionZone:   regionZoneHandler.Region.Zone,
	// 	ResourceType: call.REGIONZONE,
	// 	ResourceName: "",
	// 	CloudOSAPI:   "ListOrgZone()",
	// 	ElapsedTime:  "",
	// 	ErrorMSG:     "",
	// }

	// callLogStart := call.Start()
	// result, err := regionZoneHandler.Client.DescribeZones(request)
	// callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	// if err != nil {
	// 	callLogInfo.ErrorMSG = err.Error()
	// 	callogger.Error(call.String(callLogInfo))
	// 	return "", err
	// }
	// callogger.Info(call.String(callLogInfo))

	regionsResult, err := DescribeRegions(regionZoneHandler.Client)
	if err != nil {
		return "", err
	}

	//zoneList := map[string]*ecs.DescribeZonesResponse{}
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

		// regionStatus := GetRegionStatus(item.Status)
		// cblogger.Info("regionStatus ", regionStatus)

		// ZoneList
		var zoneInfoList []irs.ZoneInfo
		cblogger.Info("regionId ", regionId)
		zonesResult, err := DescribeZonesByRegion(regionZoneHandler.Client, regionId)
		if err != nil {
			cblogger.Debug("DescribeZone failed ", err)
		}
		for _, zone := range zonesResult.Zones.Zone {
			zoneInfo := irs.ZoneInfo{}
			zoneInfo.Name = zone.ZoneId
			zoneInfo.DisplayName = zone.LocalName
			zoneInfo.Status = GetZoneStatus("") // Zone의 상태값이 없으므로 set하지 않도록 변경.

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
				keyValue.Value = fmt.Sprintf("%v", value)
				keyValueList = append(keyValueList, keyValue)
			}
			zoneInfo.KeyValueList = keyValueList

			zoneInfoList = append(zoneInfoList, zoneInfo)
		}
		regionInfo.ZoneList = zoneInfoList
		// "ZoneType": "AvailabilityZone",
		// "LocalName": "曼谷 可用区A",
		// "ZoneId": "ap-southeast-7a",

		keyValueList := []irs.KeyValue{}
		keyValue := irs.KeyValue{}
		keyValue.Key = "RegionEndpoint"
		keyValue.Value = item.RegionEndpoint
		regionInfo.KeyValueList = keyValueList

		break
	}
	return regionInfo, nil
}

// regionList Result
// {
// 	"RequestId": "509F4448-81A7-3EB2-9D9D-2FC0BFD1AE86",
// 	"Regions": {
// 	  "Region": [
// 		{
// 		  "RegionId": "cn-qingdao",
// 		  "RegionEndpoint": "ecs.cn-qingdao.aliyuncs.com",
// 		  "LocalName": "China (Qingdao)"
// 		},
// 		{
// 		  "RegionId": "cn-beijing",
// 		  "RegionEndpoint": "ecs.cn-beijing.aliyuncs.com",
// 		  "LocalName": "China (Beijing)"
// 		},

// zoneList Result
// {
// 	"RequestId": "153128B3-EAE6-316D-A0DF-9B33D71C4C6B",
// 	"Zones": {
// 	  "Zone": [
// 		{
// 		  "ZoneId": "ap-southeast-7a",
// 		  "ZoneType": "AvailabilityZone",
// 		  "LocalName": "曼谷 可用区A",
// 		  "AvailableResourceCreation": {
// 			"ResourceTypes": [
// 			  "VSwitch", "IoOptimized", "Instance", "DedicatedHost", "Disk"
// 			]
// 		  },
// 		  "DedicatedHostGenerations": {
// 			"DedicatedHostGeneration": [
// 			  "ddh-5",
// 			  "ddh-4"
// 			]
// 		  },
// 		  "AvailableInstanceTypes": {
// 			"InstanceTypes": [
// 			  "ecs.c6e.8xlarge",...
// 			]
// 		  },
// 		  "AvailableDedicatedHostTypes": {
// 			"DedicatedHostType": [
// 			  "ddh.g6", "ddh.g5nse", "ddh.g6e", "ddh.c6"
// 			]
// 		  },
// 		  "AvailableResources": {
// 			"ResourcesInfo": [
// 			  {
// 				"InstanceGenerations": {
// 				  "supportedInstanceGeneration": [
// 					"ecs-5", "ecs-4", "ecs-6"
// 				  ]
// 				},
// 				"NetworkTypes": {
// 				  "supportedNetworkCategory": [
// 					"vpc"
// 				  ]
// 				},
// 				"IoOptimized": true,
// 				"SystemDiskCategories": {
// 				  "supportedSystemDiskCategory": [
// 					"cloud_auto", "cloud_essd"
// 				  ]
// 				},
// 				"InstanceTypes": {
// 				  "supportedInstanceType": [
// 					"ecs.c6e.8xlarge", ...
// 				  ]
// 				},
// 				"InstanceTypeFamilies": {
// 				  "supportedInstanceTypeFamily": [
// 					"ecs.gn7i",...
// 				  ]
// 				},
// 				"DataDiskCategories": {
// 				  "supportedDataDiskCategory": [
// 					"cloud_auto", "cloud_essd"
// 				  ]
// 				}
// 			  }
// 			]
// 		  },
// 		  "AvailableDiskCategories": {
// 			"DiskCategories": [
// 			  "cloud_auto", "cloud_essd"
// 			]
// 		  },
// 		  "AvailableVolumeCategories": {
// 			"VolumeCategories": []
// 		  }
// 		}
// 	  ]
// 	}
//   }
