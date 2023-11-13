// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC RegionZone Handler
//
// Created by ETRI, 2023.09.
//==================================================================================================

// RegionZoneInfo Fetch Speed Improvement and KeyValueList Omission Issue :
// https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828

package resources

import (
	// "errors"
	"sync"
	"strings"
	// "github.com/davecgh/go-spew/spew"

	// ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpRegionZoneHandler struct {
	CredentialInfo 	idrv.CredentialInfo
	RegionInfo     	idrv.RegionInfo
	VMClient        *vserver.APIClient
}

func (regionZoneHandler *NcpRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called ListRegionZone()!!")	

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListRegionZone()", "ListRegionZone()")

	ncpVpcRegionList, err := regionZoneHandler.getNcpVpcRegionList("ListRegionZone()")
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get RegionList from NCP Cloud : ", err)
		return nil, rtnErr
	}

	var regionZoneInfoList []*irs.RegionZoneInfo
	var wait sync.WaitGroup
	var zoneListError error
	for _, region := range ncpVpcRegionList {
		wait.Add(1)
		go func(region *vserver.Region) {
			defer wait.Done()
			cblogger.Info("# Search Criteria(NCP RegionCode) : ", *region.RegionCode)

			regionZoneInfo := irs.RegionZoneInfo{
				Name: 			*region.RegionCode,
				DisplayName: 	*region.RegionName,
				// KeyValueList: []irs.KeyValue{
				// 	{Key: "RegionCode", 	Value: *region.RegionCode},
				// },
			}

			ncpVpcZoneList, err := regionZoneHandler.getNcpVpcZoneList(region.RegionCode, "ListRegionZone()")
			if err != nil {
				zoneListError = err
				return
			}

			var zoneInfoList []irs.ZoneInfo
			for _, zone := range ncpVpcZoneList {
				zoneInfo := irs.ZoneInfo{
					Name: 			*zone.ZoneName,
					DisplayName: 	*zone.ZoneDescription,
					Status: 		irs.NotSupported,
					// KeyValueList: []irs.KeyValue{
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

	if zoneListError != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Zone List!!", zoneListError)
		return nil, rtnErr
	}

	return regionZoneInfoList, nil
}

func (regionZoneHandler NcpRegionZoneHandler) GetRegionZone(regionCode string) (irs.RegionZoneInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetRegionZone()!!")
	
	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionCode, "GetRegionZone()")

	if len(regionCode) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Invalid RegionCode!!", "")
		return irs.RegionZoneInfo{}, rtnErr
	}

	ncpVpcRegionList, err := regionZoneHandler.getNcpVpcRegionList("GetRegionZone()")
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get RegionList from NCP Cloud : ", err)
		return irs.RegionZoneInfo{}, rtnErr
	}

	var regionZoneInfo irs.RegionZoneInfo
	for _, region := range ncpVpcRegionList {
		if strings.EqualFold(regionCode, *region.RegionCode){
			cblogger.Info("# Search Criteria(NCP RegionCode) : ", *region.RegionCode)

			regionZoneInfo = irs.RegionZoneInfo{
				Name: 			*region.RegionCode,
				DisplayName: 	*region.RegionName,
				// KeyValueList: []irs.KeyValue{
				// 	{Key: "RegionCode", 	Value: *region.RegionCode},
				// },
			}

			ncpVpcZoneList, err := regionZoneHandler.getNcpVpcZoneList(region.RegionCode, "GetRegionZone()")
			if err != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneList from NCP Cloud : ", err)
				return irs.RegionZoneInfo{}, rtnErr
			}
	
			var zoneInfoList []irs.ZoneInfo
			for _, zone := range ncpVpcZoneList {
				zoneInfo := irs.ZoneInfo{
					Name: 			*zone.ZoneName,
					DisplayName: 	*zone.ZoneDescription,
					Status: 		irs.NotSupported,
					// KeyValueList: []irs.KeyValue{
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
	cblogger.Info("NCP VPC Cloud Driver: called ListOrgRegion()!!")	

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListOrgRegion()", "ListOrgRegion()")

	// To return the results with a style similar to other CSPs.
	type Regions struct {
		RegionList 	[]*vserver.Region // Must be a capital letter!!
	}

	ncpVpcRegionList, err := regionZoneHandler.getNcpVpcRegionList("ListOrgRegion()")
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get RegionList from NCP Cloud : ", err)
		return "", rtnErr
	}
	ncpRegionList := Regions{
		RegionList: ncpVpcRegionList,
	}
	jsonString, cvtErr := ConvertJsonString(ncpRegionList)
	if cvtErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert the RegionList to Json format string.", cvtErr)
		return "", rtnErr
	}
	return jsonString, cvtErr
}

func (regionZoneHandler *NcpRegionZoneHandler) ListOrgZone() (string, error) {
	cblogger.Info("NCP VPC Cloud Driver: called ListOrgZone()!!")	

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListOrgZone()", "ListOrgZone()")

	if len(regionZoneHandler.RegionInfo.Region) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Invalid Region Info!!", "")
		return "", rtnErr
	}

	// To return the results with a style similar to other CSPs.
	type Zones struct {
		ZoneList 	[]*vserver.Zone // Must be a capital letter!!
	}

	ncpVpcZoneList, err := regionZoneHandler.getNcpVpcZoneList(&regionZoneHandler.RegionInfo.Region, "ListOrgZone()")
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneList from NCP Cloud : ", err)
		return "", rtnErr
	}

	ncpZoneList := Zones{
		ZoneList: ncpVpcZoneList,
	}
	jsonString, cvtErr := ConvertJsonString(ncpZoneList)
	if cvtErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert the ZoneList to Json format string.", cvtErr)
		return "", rtnErr
	}
	return jsonString, cvtErr
}

func (regionZoneHandler *NcpRegionZoneHandler) getNcpVpcRegionList(callLogfunc string) ([]*vserver.Region, error) {
	cblogger.Info("NCP VPC Cloud Driver: called getNcpVpcRegionList()!!")	

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, callLogfunc, callLogfunc)

	regionListReq := vserver.GetRegionListRequest{}
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
	return regionListResult.RegionList, nil
}

func (regionZoneHandler *NcpRegionZoneHandler) getNcpVpcZoneList(regionCode *string, callLogfunc string) ([]*vserver.Zone, error) {
	cblogger.Info("NCP VPC Cloud Driver: called getNcpVpcZoneList()!!")	

	InitLog()
	callLogInfo := GetCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, *regionCode, callLogfunc)

	zoneListReq := vserver.GetZoneListRequest{RegionCode: regionCode}
	callLogStart := call.Start()
	zoneListResult, err := regionZoneHandler.VMClient.V2Api.GetZoneList(&zoneListReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneList from NCP Cloud : ", err)
		return nil, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	
	if len(zoneListResult.ZoneList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Zone Info.", "")
		return nil, rtnErr
	} else {
		cblogger.Infof("# Supported Zone count [%s] : [%d]", *regionCode, len(zoneListResult.ZoneList))
		// spew.Dump(zoneListResult)
	}
	return zoneListResult.ZoneList, nil
}
