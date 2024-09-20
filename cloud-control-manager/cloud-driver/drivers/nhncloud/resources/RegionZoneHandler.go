// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NHN Cloud RegionZone Handler
//
// Created by ETRI, 2023.09.
//==================================================================================================

package resources

import (
	"sort"
	"strings"
	"sync"
	// "errors"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	ostack "github.com/cloud-barista/nhncloud-sdk-go/openstack"
	az "github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/availabilityzones"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ##### (Note) NHN Cloud only provides detailed Zone information with the API, but unlike other CSPs, the API Endpoint is different for each Region, so when calling with API, only the zone information of the Region comes out without the Region information.
// NHN Region Info. Ref) https://docs.nhncloud.com/ko/Compute/Compute/ko/identity-api/
// KeyValueList Omission Issue : https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828

type NhnRegionInfo struct {
	RegionCode string
	RegionName string
}

// As Constant Variables
func getSupportedRegions() []NhnRegionInfo {
	regionInfoList := []NhnRegionInfo{
		{RegionCode: "KR1",
			RegionName: "한국(판교)",
		},
		{RegionCode: "KR2",
			RegionName: "한국(평촌)",
		},
		{RegionCode: "JP1",
			RegionName: "일본",
		},
	}
	return regionInfoList
}

type NhnCloudRegionZoneHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *nhnsdk.ServiceClient
}

func (regionZoneHandler *NhnCloudRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListRegionZone()!!")
	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListRegionZone()", "ListRegionZone()")

	nhnRegionList := getSupportedRegions()
	// cblogger.Infof("# nhnRegionlist : [%d]", len(nhnRegionList))

	// ### Even though NHN Cloud does not provide Region info., to get the Zonelist of 'All' Region.
	var regionZoneInfoList []*irs.RegionZoneInfo
	var wait sync.WaitGroup
	var zoneInfoListError error
	for _, regionInfo := range nhnRegionList {
		wait.Add(1)
		go func(regionInfo NhnRegionInfo) {
			defer wait.Done()
			cblogger.Info("# NHN RegionCode : ", regionInfo.RegionCode)

			regionZoneInfo := irs.RegionZoneInfo{
				Name:        regionInfo.RegionCode,
				DisplayName: regionInfo.RegionName,
				// KeyValueList: []irs.KeyValue{
				// 	{Key: "RegionCode", 	Value: regionInfo.RegionCode},
				// },
			}

			zoneInfoList, err := regionZoneHandler.getZoneInfoList(regionInfo.RegionCode)
			if err != nil {
				zoneInfoListError = err
				return
			}
			regionZoneInfo.ZoneList = zoneInfoList
			regionZoneInfoList = append(regionZoneInfoList, &regionZoneInfo)
		}(regionInfo)

	}
	wait.Wait()

	if zoneInfoListError != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneInfoList!!", zoneInfoListError)
		return nil, rtnErr
	}

	sort.Slice(regionZoneInfoList, func(i, j int) bool {
		return strings.Compare(regionZoneInfoList[i].Name, regionZoneInfoList[j].Name) < 0
	})

	return regionZoneInfoList, nil
}

func (regionZoneHandler NhnCloudRegionZoneHandler) GetRegionZone(regionCode string) (irs.RegionZoneInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetRegionZone()!!")
	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionCode, "GetRegionZone()")

	nhnRegionList := getSupportedRegions()
	var regionZoneInfo irs.RegionZoneInfo
	for _, regionInfo := range nhnRegionList {
		// cblogger.Info("# NCP RegionCode : ", regionInfo.RegionCode)

		if strings.EqualFold(regionCode, regionInfo.RegionCode) {
			regionZoneInfo = irs.RegionZoneInfo{
				Name:        regionInfo.RegionCode,
				DisplayName: regionInfo.RegionName,
				// KeyValueList: []irs.KeyValue{
				// 	{Key: "RegionCode", 	Value: regionInfo.RegionCode},
				// },
			}
		}
	}

	// If there is no Region information in the driver, ...
	if strings.EqualFold(regionZoneInfo.DisplayName, "") {
		regionZoneInfo = irs.RegionZoneInfo{
			Name:        regionCode,
			DisplayName: "",
			// KeyValueList: []irs.KeyValue{
			// 	{Key: "RegionCode", 	Value: regionCode},
			// },
		}
	}

	zoneInfoList, err := regionZoneHandler.getZoneInfoList(regionCode)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneInfoList : ", err)
		return irs.RegionZoneInfo{}, rtnErr
	}
	regionZoneInfo.ZoneList = zoneInfoList
	return regionZoneInfo, nil
}

func (regionZoneHandler *NhnCloudRegionZoneHandler) ListOrgRegion() (string, error) {
	cblogger.Info("NHN Cloud Driver: called ListOrgRegion()!!")
	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "ListOrgRegion()", "ListOrgRegion()")

	// To return the results with a style similar to other CSPs.
	type Regions struct {
		RegionList []NhnRegionInfo
	}

	nhnRegionList := getSupportedRegions()
	regionList := Regions{
		RegionList: nhnRegionList,
	}
	jsonString, err := convertJsonString(regionList)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert to Json String : ", err)
		return "", rtnErr
	}
	return jsonString, nil
}

func (regionZoneHandler *NhnCloudRegionZoneHandler) ListOrgZone() (string, error) {
	cblogger.Info("NHN Cloud Driver: called ListOrgZone()!!")

	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionZoneHandler.RegionInfo.Region, "ListOrgZone()")

	// To return the results with a style similar to other CSPs.
	type Zones struct {
		ZoneList []az.AvailabilityZone
	}

	nhnZoneList, err := regionZoneHandler.getNhnZoneList(regionZoneHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneInfoList : ", err)
		return "", rtnErr
	}
	zoneList := Zones{
		ZoneList: nhnZoneList,
	}
	jsonString, err := convertJsonString(zoneList)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert to Json String : ", err)
		return "", rtnErr
	}
	return jsonString, nil
}

func (regionZoneHandler NhnCloudRegionZoneHandler) getZoneInfoList(regionCode string) ([]irs.ZoneInfo, error) {
	cblogger.Info("NHN Cloud Driver: called getZoneInfoList()!!")
	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionCode, "getZoneInfoList()")

	if strings.EqualFold(regionCode, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid RegionCode!!", "")
		return nil, rtnErr
	}

	nhnZoneList, err := regionZoneHandler.getNhnZoneList(regionCode)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get ZoneInfoList : ", err)
		return nil, rtnErr
	}

	var zoneInfoList []irs.ZoneInfo
	for _, zone := range nhnZoneList {
		var zoneStatus irs.ZoneStatus
		if zone.ZoneState.Available {
			zoneStatus = irs.ZoneAvailable
		} else {
			zoneStatus = irs.ZoneUnavailable
		}

		zoneInfo := irs.ZoneInfo{
			Name:        zone.ZoneName,
			DisplayName: zone.ZoneName,
			Status:      zoneStatus,
			// KeyValueList: []irs.KeyValue{
			// 	{Key: "ZoneCode", 	Value: zone.ZoneName},
			// },
		}
		zoneInfoList = append(zoneInfoList, zoneInfo)
	}

	sort.Slice(zoneInfoList, func(i, j int) bool {
		return strings.Compare(zoneInfoList[i].Name, zoneInfoList[j].Name) < 0
	})

	return zoneInfoList, nil
}

func (regionZoneHandler NhnCloudRegionZoneHandler) getNhnZoneList(regionCode string) ([]az.AvailabilityZone, error) {
	cblogger.Info("NHN Cloud Driver: called getNhnZoneList()!!")
	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, regionCode, "getNhnZoneList()")

	if strings.EqualFold(regionCode, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid RegionCode!!", "")
		return nil, rtnErr
	}

	regionInfo := idrv.RegionInfo{
		Region: regionCode,
	}
	connInfo := idrv.ConnectionInfo{
		CredentialInfo: regionZoneHandler.CredentialInfo,
		RegionInfo:     regionInfo,
	}
	vmClient, err := regionZoneHandler.getNhnVMClient(connInfo)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get VMClient : ", err)
		return nil, rtnErr
	}
	callLogStart := call.Start()
	allPages, err := az.List(vmClient).AllPages()
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Zone Pages from NHN Cloud : ", err)
		return nil, rtnErr
	}
	nhnZoneList, err := az.ExtractAvailabilityZones(allPages)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Zone List from NHN Cloud : ", err)
		return nil, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	// cblogger.Infof("# Count : [%d]", len(nhnZoneList))
	// spew.Dump(nhnZoneList)
	return nhnZoneList, nil
}

func (regionZoneHandler NhnCloudRegionZoneHandler) getNhnVMClient(connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
	cblogger.Info("NHN Cloud Driver: called getNhnVMClient()!!")
	callLogInfo := getCallLogScheme(regionZoneHandler.RegionInfo.Zone, call.REGIONZONE, "getNhnVMClient()", "getNhnVMClient()")

	authOpts := nhnsdk.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.TenantId, // Caution : TenantID spelling for SDK
	}

	if strings.EqualFold(authOpts.IdentityEndpoint, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid IdentityEndpoint!!", "")
		return nil, rtnErr
	}

	providerClient, err := ostack.AuthenticatedClient(authOpts)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the ProviderClient Info : ", err)
		return nil, rtnErr
	}
	vmClient, err := ostack.NewComputeV2(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get VMClient Info : ", err)
		return nil, rtnErr
	}
	return vmClient, err
}
