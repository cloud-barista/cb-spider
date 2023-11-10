// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP Classic RegionZone Handler
//
// Created by ETRI, 2023.09.
//==================================================================================================

package resources

import (
	"fmt"
	// "errors"
	"sync"
	"strings"
	// "github.com/davecgh/go-spew/spew"

	// ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// KeyValueList Omission Issue : https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828

type NcpRegionZoneHandler struct {
	CredentialInfo 	idrv.CredentialInfo
	RegionInfo     	idrv.RegionInfo
	VMClient        *server.APIClient
}

func (regionZoneHandler *NcpRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called ListRegionZone()!!")	

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListRegionZone()", "ListRegionZone()")

	regionListReq := server.GetRegionListRequest{}
	callLogStart := call.Start()
	regionListResult, err := regionZoneHandler.VMClient.V2Api.GetRegionList(&regionListReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get RegionList from NCP Cloud : ", err)
		return nil, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(regionListResult.RegionList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Region Info.", "")
		return nil, rtnErr
	} else {
		cblogger.Infof("# Supported Region count : [%d]", len(regionListResult.RegionList))
		// spew.Dump(regionListResult)
	}

	var regionZoneInfoList []*irs.RegionZoneInfo
	var wait sync.WaitGroup
	var zoneListErr error
	var zoneExistErr error
	for _, region := range regionListResult.RegionList {
		wait.Add(1)
		go func(region *server.Region) {
			defer wait.Done()
			cblogger.Info("# Search Criteria(NCP RegionCode) : ", *region.RegionCode)

			regionZoneInfo := irs.RegionZoneInfo{
				Name: 			*region.RegionCode,
				DisplayName: 	*region.RegionName,
				// KeyValueList: []irs.KeyValue{
				// 	{Key: "RegionNo", 		Value: *region.RegionNo},
				// 	{Key: "RegionCode", 	Value: *region.RegionCode},
				// },
			}
			zoneListReq := server.GetZoneListRequest{
				RegionNo: 	region.RegionNo,
				//RegionNo: nil, // Caution!! : If look up like this, only Korean two zones will come out.
			}
			callLogStart := call.Start()
			zoneListResult, err := regionZoneHandler.VMClient.V2Api.GetZoneList(&zoneListReq)
			if err != nil {
				zoneListErr = err
				return
			}
			LoggingInfo(callLogInfo, callLogStart)
			
			if len(zoneListResult.ZoneList) < 1 {
				zoneExistErr = fmt.Errorf("Failed to Find Any Zone Info!!")
				return
			} else {
				cblogger.Infof("# Supported Zone count : [%d]", len(zoneListResult.ZoneList))
				// spew.Dump(zoneListResult)
			}

			var zoneInfoList []irs.ZoneInfo
			for _, zone := range zoneListResult.ZoneList {
				zoneInfo := irs.ZoneInfo{
					Name: 			*zone.ZoneName,
					DisplayName: 	*zone.ZoneDescription,
					Status: 		irs.NotSupported,				
					// KeyValueList: []irs.KeyValue{
					// 	{Key: "ZoneNo", 	Value: *zone.ZoneNo},
					// 	{Key: "ZoneCode", 	Value: *zone.ZoneCode},
					// },
				}
				zoneInfoList = append(zoneInfoList, zoneInfo)		
			}
			regionZoneInfo.ZoneList = zoneInfoList
			regionZoneInfoList = append(regionZoneInfoList, &regionZoneInfo)
		}(region)
	}
	wait.Wait()

	if zoneListErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Zone List!!", zoneListErr)
		return nil, rtnErr
	}
	if zoneExistErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Zone Info.", zoneExistErr)
		return nil, rtnErr
	}

	return regionZoneInfoList, nil
}

func (regionZoneHandler NcpRegionZoneHandler) GetRegionZone(regionCode string) (irs.RegionZoneInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetRegionZone()!!")

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionCode, "GetRegionZone()")

	if len(regionCode) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "The RegionCode is Empty!!", "")
		return irs.RegionZoneInfo{}, rtnErr
	}

	regionListReq := server.GetRegionListRequest{}
	callLogStart := call.Start()
	regionListResult, err := regionZoneHandler.VMClient.V2Api.GetRegionList(&regionListReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get RegionList from NCP Cloud : ", err)
		return irs.RegionZoneInfo{}, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(regionListResult.RegionList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Region Info.", "")
		return irs.RegionZoneInfo{}, rtnErr
	} else {
		cblogger.Infof("# Supported Region count : [%d]", len(regionListResult.RegionList))
		// spew.Dump(regionListResult)
	}

	var regionZoneInfo irs.RegionZoneInfo
	for _, region := range regionListResult.RegionList {
		if strings.EqualFold(regionCode, *region.RegionCode){
			cblogger.Info("# Search Criteria(NCP RegionCode) : ", *region.RegionCode)

			regionZoneInfo = irs.RegionZoneInfo{
				Name: 			*region.RegionCode,
				DisplayName: 	*region.RegionName,	
				// KeyValueList: []irs.KeyValue{
				// 	{Key: "RegionNo", 		Value: *region.RegionNo},
				// 	{Key: "RegionCode", 	Value: *region.RegionCode},
				// },
			}
			zoneListReq := server.GetZoneListRequest{
				RegionNo: 	region.RegionNo,
				//RegionNo: nil, //CAUTION!! : 이렇게 조회하면 zone이 한국 zone 두개만 나옴.
			}
			callLogStart := call.Start()
			zoneListResult, err := regionZoneHandler.VMClient.V2Api.GetZoneList(&zoneListReq)
			if err != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneList from NCP Cloud : ", err)
				return irs.RegionZoneInfo{}, rtnErr
			}
			LoggingInfo(callLogInfo, callLogStart)
			
			if len(zoneListResult.ZoneList) < 1 {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Zone Info.", "")
				return irs.RegionZoneInfo{}, rtnErr
			} else {
				cblogger.Infof("# Supported Zone count : [%d]", len(zoneListResult.ZoneList))
				// spew.Dump(zoneListResult)
			}
	
			var zoneInfoList []irs.ZoneInfo
			for _, zone := range zoneListResult.ZoneList {
				zoneInfo := irs.ZoneInfo{
					Name: 			*zone.ZoneName,
					DisplayName: 	*zone.ZoneDescription,
					Status: 		irs.NotSupported,	
					// KeyValueList: []irs.KeyValue{
					// 	{Key: "ZoneNo", 	Value: *zone.ZoneNo},
					// 	{Key: "ZoneCode", 	Value: *zone.ZoneCode},
					// },
				}
				zoneInfoList = append(zoneInfoList, zoneInfo)		
			}
			regionZoneInfo.ZoneList = zoneInfoList
		}
	}
	return regionZoneInfo, nil
}

func (regionZoneHandler *NcpRegionZoneHandler) ListOrgRegion() (string, error) {
	cblogger.Info("NCP Classic Cloud Driver: called ListOrgRegion()!!")	

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListOrgRegion()", "ListOrgRegion()")

	regionListReq := server.GetRegionListRequest{}
	callLogStart := call.Start()
	regionListResult, err := regionZoneHandler.VMClient.V2Api.GetRegionList(&regionListReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get RegionList from NCP Cloud : ", err)
		return "", rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(regionListResult.RegionList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Region Info.", "")
		return "", rtnErr
	} else {
		cblogger.Infof("# Supported Region count : [%d]", len(regionListResult.RegionList))
		// spew.Dump(regionListResult)
	}

	jsonString, convertErr := ConvertJsonString(regionListResult)
	if convertErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert the Region List to Json format string.", convertErr)
		return "", rtnErr
	}
	return jsonString, convertErr
}

func (regionZoneHandler *NcpRegionZoneHandler) ListOrgZone() (string, error) {
	cblogger.Info("NCP Classic Cloud Driver: called ListOrgZone()!!")	

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionZoneHandler.RegionInfo.Region, "ListOrgZone()")

	if len(regionZoneHandler.RegionInfo.Region) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "RegionInfo.Region is Empty!!", "")
		return "", rtnErr
	}

	vmHandler := NcpVMHandler{
		CredentialInfo: 	regionZoneHandler.CredentialInfo,
		RegionInfo:     	regionZoneHandler.RegionInfo,
		VMClient:         	regionZoneHandler.VMClient,
	}
	regionNo, err := vmHandler.GetRegionNo(regionZoneHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Region No of the Region Code : ", err)
		return "", rtnErr
	}
	zoneListReq := server.GetZoneListRequest{
		RegionNo: 	regionNo,
		//RegionNo: nil, //CAUTION!! : 이렇게 조회하면 zone이 한국 zone 두개만 나옴.
	}
	callLogStart := call.Start()
	zoneListResult, err := regionZoneHandler.VMClient.V2Api.GetZoneList(&zoneListReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneList from NCP Cloud : ", err)
		return "", rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(zoneListResult.ZoneList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Zone Info.", "")
		return "", rtnErr
	} else {
		cblogger.Infof("# Supported Zone count : [%d]", len(zoneListResult.ZoneList))
		// spew.Dump(zoneListResult)
	}

	jsonString, convertErr := ConvertJsonString(zoneListResult)
	if convertErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert the Zone List to Json format string.", convertErr)
		return "", rtnErr
	}
	return jsonString, convertErr
}
