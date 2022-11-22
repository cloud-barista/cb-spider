// Cloud Driver Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.12.

package clouddriverhandler

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
	icbs "github.com/cloud-barista/cb-store/interfaces"

	"fmt"
	"strings"
)

/*
func ListCloudDriver() []string {
        var cloudDriverList []string
        // @todo get list from storage
        return cloudDriverList
}
*/

// 1. get the ConnectionConfig Info
// 2. get the driver info
// 3. get CloudDriver
func GetCloudDriver(cloudConnectName string) (idrv.CloudDriver, error) {
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
	if err != nil {
		return nil, err
	}

	cldDrvInfo, err := dim.GetCloudDriver(cccInfo.DriverName)
	if err != nil {
		return nil, err
	}

	return getCloudDriver(*cldDrvInfo)
}

// 1. get credential info
// 2. get region info
// 3. get CloudConneciton
func GetCloudConnection(cloudConnectName string) (icon.CloudConnection, error) {
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
	if err != nil {
		return nil, err
	}

	cldDriver, err := GetCloudDriver(cloudConnectName)
	if err != nil {
		return nil, err
	}

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return nil, err
	}

	rgnInfo, err := rim.GetRegion(cccInfo.RegionName)
	if err != nil {
		return nil, err
	}

	regionName, zoneName, err := getRegionNameByRegionInfo(rgnInfo)
	if err != nil {
		return nil, err
	}

	connectionInfo := idrv.ConnectionInfo{ // @todo powerkim
		CredentialInfo: idrv.CredentialInfo{
			ClientId:         getValue(crdInfo.KeyValueInfoList, "ClientId"),
			ClientSecret:     getValue(crdInfo.KeyValueInfoList, "ClientSecret"),
			TenantId:         getValue(crdInfo.KeyValueInfoList, "TenantId"),
			SubscriptionId:   getValue(crdInfo.KeyValueInfoList, "SubscriptionId"),
			IdentityEndpoint: getValue(crdInfo.KeyValueInfoList, "IdentityEndpoint"),
			Username:         getValue(crdInfo.KeyValueInfoList, "Username"),
			Password:         getValue(crdInfo.KeyValueInfoList, "Password"),
			DomainName:       getValue(crdInfo.KeyValueInfoList, "DomainName"),
			ProjectID:        getValue(crdInfo.KeyValueInfoList, "ProjectID"),
			AuthToken:        getValue(crdInfo.KeyValueInfoList, "AuthToken"),
			ClientEmail:      getValue(crdInfo.KeyValueInfoList, "ClientEmail"),
			PrivateKey:       getValue(crdInfo.KeyValueInfoList, "PrivateKey"),
			Host:             getValue(crdInfo.KeyValueInfoList, "Host"),
			APIVersion:       getValue(crdInfo.KeyValueInfoList, "APIVersion"),
			MockName:         getValue(crdInfo.KeyValueInfoList, "MockName"),
			ApiKey:           getValue(crdInfo.KeyValueInfoList, "ApiKey"),
			ClusterId:        getValue(crdInfo.KeyValueInfoList, "ClusterId"),
			ConnectionName:   cloudConnectName,
		},
		RegionInfo: idrv.RegionInfo{ // @todo powerkim
			Region:        regionName,
			Zone:          zoneName,
			ResourceGroup: getValue(rgnInfo.KeyValueInfoList, "ResourceGroup"),
		},
	}

	cldConnection, err := cldDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}

	return cldConnection, nil
}

func GetProviderNameByConnectionName(cloudConnectName string) (string, error) {
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
	if err != nil {
		return "", err
	}

	rgnInfo, err := rim.GetRegion(cccInfo.RegionName)
	if err != nil {
		return "", err
	}

	return rgnInfo.ProviderName, nil
}

func GetRegionNameByConnectionName(cloudConnectName string) (string, string, error) {
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
	if err != nil {
		return "", "", err
	}

	rgnInfo, err := rim.GetRegion(cccInfo.RegionName)
	if err != nil {
		return "", "", err
	}

	return getRegionNameByRegionInfo(rgnInfo)
}

func getRegionNameByRegionInfo(rgnInfo *rim.RegionInfo) (string, string, error) {

	// @todo should move KeyValueList into XXXDriver.go, powerkim
	var regionName string
	var zoneName string
	switch strings.ToUpper(rgnInfo.ProviderName) {
	case "AZURE":
		regionName = getValue(rgnInfo.KeyValueInfoList, "location")
	case "AWS", "ALIBABA", "GCP", "TENCENT", "IBM", "NCP", "NCPVPC", "KTCLOUD", "NHNCLOUD":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
		zoneName = getValue(rgnInfo.KeyValueInfoList, "Zone")
	case "OPENSTACK", "CLOUDIT", "DOCKER", "CLOUDTWIN", "MOCK", "MINI":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	default:
		errmsg := rgnInfo.ProviderName + " is not a valid ProviderName!!"
		return "", "", fmt.Errorf(errmsg)
	}

	return regionName, zoneName, nil
}

func getValue(keyValueInfoList []icbs.KeyValue, key string) string {
	for _, kv := range keyValueInfoList {
		if kv.Key == key {
			return kv.Value
		}
	}
	return "Not set"
}
