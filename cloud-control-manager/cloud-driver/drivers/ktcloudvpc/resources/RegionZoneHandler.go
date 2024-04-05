// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud VPC RegionZone Handler
//
// Created by ETRI, 2023.10.
//==================================================================================================

package resources

import (
	"strings"
	"errors"
	// "github.com/davecgh/go-spew/spew"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KtVpcZone struct {
    ZoneCode 	string
    ZoneName	string
}

type KtVpcRegion struct {
    RegionCode 	string
    RegionName	string
	ZoneList 	[]KtVpcZone
}

// As Constant Variables
func getSupportedRegionZones() []KtVpcRegion {
	regionList := []KtVpcRegion {
		{	RegionCode: 	"KR1",
			RegionName: 	"서울",
			ZoneList: []KtVpcZone {
				{	ZoneCode: 	"DX-M1",
					ZoneName: 	"목동-1",
				},
			},
		},
	}
	return regionList
}

type KTVpcRegionZoneHandler struct {
	RegionInfo    	idrv.RegionInfo
}

func (regionZoneHandler *KTVpcRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListRegionZone()!!")

	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListRegionZone()", "ListRegionZone()")

	ktVpcRegionZoneList := getSupportedRegionZones()
	if len(ktVpcRegionZoneList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Region Info.", nil)
		return nil, rtnErr
	}

	var regionZoneInfoList []*irs.RegionZoneInfo
	for _, region := range ktVpcRegionZoneList {
			cblogger.Info("# KT RegionCode : ", region.RegionCode)

			regionZoneInfo := irs.RegionZoneInfo{
				Name: 			region.RegionCode,
				DisplayName: 	region.RegionName,
			}

			zoneInfoList, err := regionZoneHandler.getZoneInfoList(region.RegionCode)
			if err != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Get KT Cloud Zone Info : ", err)
				return nil, rtnErr
			}
			if len(zoneInfoList) < 1 {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Zone Info.", nil)
				return nil, rtnErr
			}
			regionZoneInfo.ZoneList = zoneInfoList
			regionZoneInfoList = append(regionZoneInfoList, &regionZoneInfo)
	}
	return regionZoneInfoList, nil
}

func (regionZoneHandler KTVpcRegionZoneHandler) GetRegionZone(regionCode string) (irs.RegionZoneInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetRegionZone()!!")	

	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionCode, "GetRegionZone()")

	if len(regionCode) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "The RegionCode is Empty!!", nil)
		return irs.RegionZoneInfo{}, rtnErr
	}

	ktVpcRegionZoneList := getSupportedRegionZones()
	if len(ktVpcRegionZoneList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Region Info.", nil)
		return irs.RegionZoneInfo{}, rtnErr
	}

	var regionZoneInfo irs.RegionZoneInfo
	for _, region := range ktVpcRegionZoneList {
		if strings.EqualFold(regionCode, region.RegionCode) {
			regionZoneInfo = irs.RegionZoneInfo {
				Name: 			region.RegionCode,
				DisplayName: 	region.RegionName,
			}
		}
	}

	zoneInfoList, err := regionZoneHandler.getZoneInfoList(regionCode)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneInfoList : ", err)
		return irs.RegionZoneInfo{}, rtnErr
	}
	if len(zoneInfoList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Zone Info.", nil)
		return irs.RegionZoneInfo{}, rtnErr
	}
	regionZoneInfo.ZoneList = zoneInfoList
	return regionZoneInfo, nil
}

func (regionZoneHandler *KTVpcRegionZoneHandler) ListOrgRegion() (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListOrgRegion()!!")

	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListOrgRegion()", "ListOrgRegion()")

	type Region struct {
		RegionCode 	string
		RegionName	string
	}

	ktVpcRegionZoneList := getSupportedRegionZones()
	if len(ktVpcRegionZoneList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Region Info.", nil)
		return "", rtnErr
	}

	var regionInfoList []Region
	for _, region := range ktVpcRegionZoneList {
			cblogger.Info("# KT RegionCode : ", region.RegionCode)

			regionInfo := Region{
				RegionCode: region.RegionCode,
				RegionName: region.RegionName,
			}
			regionInfoList = append(regionInfoList, regionInfo)
	}

	// To return the results with a style similar to other CSPs.
	type Regions struct {
		RegionList 	[]Region
	}

	regionList := Regions{
		RegionList: regionInfoList,
	}
	jsonString, err := convertJsonString(regionList)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert to Json String : ", err)
		return "", rtnErr
	}
	return jsonString, nil
}

func (regionZoneHandler *KTVpcRegionZoneHandler) ListOrgZone() (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListOrgZone()!!")	

	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionZoneHandler.RegionInfo.Region, "ListOrgZone()")

	if strings.EqualFold(regionZoneHandler.RegionInfo.Region, "") {
		err := errors.New("'regionZoneHandler.RegionInfo.Region' invalid")
		rtnErr := logAndReturnError(callLogInfo, "Invalid RegionCode!! ", err)
		return "", rtnErr
	}

	// To return the results with a style similar to other CSPs.
	type Zones struct {
		ZoneList 	[]KtVpcZone
	}

	ktVpcRegionZoneList := getSupportedRegionZones()
	if len(ktVpcRegionZoneList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Region Info.", nil)
		return "", rtnErr
	}

	var ktVpcZoneList []KtVpcZone
	for _, region := range ktVpcRegionZoneList {
		if strings.EqualFold(regionZoneHandler.RegionInfo.Region, region.RegionCode) {
			if len(region.ZoneList) < 1 {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Zone Info in the Region Info.", nil)
				return "", rtnErr
			}
			ktVpcZoneList = region.ZoneList
		}
	}

	zoneList := Zones{
		ZoneList: ktVpcZoneList,
	}
	jsonString, err := convertJsonString(zoneList)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert to Json String : ", err)
		return "", rtnErr
	}
	return jsonString, nil
}

func (regionZoneHandler KTVpcRegionZoneHandler) getZoneInfoList(regionCode string) ([]irs.ZoneInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called getZoneInfoList()!!")	

	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionCode, "getZoneInfoList()")

	if strings.EqualFold(regionCode, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid RegionCode!!", nil)
		return nil, rtnErr
	}

	ktVpcRegionZoneList := getSupportedRegionZones()
	if len(ktVpcRegionZoneList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Region Info.", nil)
		return nil, rtnErr
	}

	var zoneInfoList []irs.ZoneInfo
	for _, region := range ktVpcRegionZoneList {
		if strings.EqualFold(regionCode, region.RegionCode) {
			cblogger.Info("# KT RegionCode : ", region.RegionCode)
			if len(region.ZoneList) < 1 {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Find Any Zone Info in the Region Info.", nil)
				return nil, rtnErr
			}

			for _, zone := range region.ZoneList {
				cblogger.Info("# KT ZoneCode : ", zone.ZoneCode)
	
				zoneInfo := irs.ZoneInfo{
					Name: 			zone.ZoneCode,
					DisplayName: 	zone.ZoneName,
					Status:			irs.NotSupported,
				}	
				zoneInfoList = append(zoneInfoList, zoneInfo)
			}
		}
	}
	return zoneInfoList, nil
}
