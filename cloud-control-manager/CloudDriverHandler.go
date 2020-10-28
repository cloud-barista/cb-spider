// Cloud Driver Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.09.

package clouddriverhandler

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	alibabadrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba"
	awsdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws"
	azuredrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure"
	clouditdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit"
	dockerdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/docker"
	gcpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp"
	openstackdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"

	//	cloudtwindrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudtwin"
	ncpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp" // NCP

	icbs "github.com/cloud-barista/cb-store/interfaces"

	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"

	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"

	//"encoding/json"
	"fmt"
	//"net/http"
	"os"
	"plugin"
	"strings"
)

var CIM_RESTSERVER = "http://localhost:1024"
var cblog *logrus.Logger

func init() {
	//CIM_RESTSERVER = "http://localhost:1024"
	cblog = config.Cblogger
}

/*
func ListCloudDriver() []string {
	var cloudDriverList []string
	// @todo get list from storage
	return cloudDriverList
}
*/

// 1. get the ConnectionConfig Info
// 2. get the driver info
// 3. load driver library
// 4. get CloudDriver
func GetCloudDriver(cloudConnectName string) (idrv.CloudDriver, error) {
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
	if err != nil {
		return nil, err
	}

	cldDrvInfo, err := dim.GetCloudDriver(cccInfo.DriverName)
	if err != nil {
		return nil, err
	}

	pluginSW := os.Getenv("PLUGIN_SW")
	if strings.ToUpper(pluginSW) == "OFF" {
		return getStaticCloudDriver(*cldDrvInfo)
	} else {
		return getCloudDriver(*cldDrvInfo)
	}
}

// 1. get credential info
// 2. get region info
// 3. get CloudConneciton
func GetCloudConnection(cloudConnectName string) (icon.CloudConnection, error) {
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
	if err != nil {
		return nil, err
	}

	cldDrvInfo, err := dim.GetCloudDriver(cccInfo.DriverName)
	if err != nil {
		return nil, err
	}

	pluginSW := os.Getenv("PLUGIN_SW")
	var cldDriver idrv.CloudDriver
	if strings.ToUpper(pluginSW) == "OFF" {
		cldDriver, err = getStaticCloudDriver(*cldDrvInfo)
	} else {
		cldDriver, err = getCloudDriver(*cldDrvInfo)
	}
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

	//cblog.Info(cldDriver)
	//cblog.Info(crdInfo)
	//cblog.Info(rgnInfo)

	regionName, zoneName, err := GetRegionNameByRegionInfo(rgnInfo)
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
			AccessKeyID:      getValue(crdInfo.KeyValueInfoList, "AccessKeyID"), // NCP
			SecretKey:        getValue(crdInfo.KeyValueInfoList, "SecretKey"),   // NCP
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

func getValue(keyValueInfoList []icbs.KeyValue, key string) string {
	for _, kv := range keyValueInfoList {
		if kv.Key == key {
			return kv.Value
		}
	}
	return "Not set"
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

	return GetRegionNameByRegionInfo(rgnInfo)
}

func GetRegionNameByRegionInfo(rgnInfo *rim.RegionInfo) (string, string, error) {

	// @todo should move KeyValueList into XXXDriver.go, powerkim
	var regionName string
	var zoneName string
	switch strings.ToUpper(rgnInfo.ProviderName) {
	case "AZURE":
		regionName = getValue(rgnInfo.KeyValueInfoList, "location")
	case "AWS":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
		zoneName = getValue(rgnInfo.KeyValueInfoList, "Zone")
	case "ALIBABA":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
		zoneName = getValue(rgnInfo.KeyValueInfoList, "Zone")
	case "GCP":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
		zoneName = getValue(rgnInfo.KeyValueInfoList, "Zone")
	case "OPENSTACK":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	case "CLOUDTWIN":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	case "CLOUDIT":
		// Cloudit do not use Region, But set default @todo 2019.10.28. by powerkim.
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	case "DOCKER":
		// docker do not use Region, But set default @todo 2020.05.06. by powerkim.
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	case "NCP": // NCP
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region") // NCP
	default:
		errmsg := rgnInfo.ProviderName + " is not a valid ProviderName!!"
		return "", "", fmt.Errorf(errmsg)
	}

	return regionName, zoneName, nil
}

func getCloudDriver(cldDrvInfo dim.CloudDriverInfo) (idrv.CloudDriver, error) {
	// $CBSPIDER_ROOT/cloud-driver-libs/*
	cbspiderRoot := os.Getenv("CBSPIDER_ROOT")
	if cbspiderRoot == "" {
		cblog.Error("$CBSPIDER_ROOT is not set!!")
		os.Exit(1)
	}
	driverLibPath := cbspiderRoot + "/cloud-driver-libs/"

	driverFile := cldDrvInfo.DriverLibFileName // ex) "aws-test-driver-v0.5.so"
	if driverFile == "" {
		return nil, fmt.Errorf("%q: driver library file can't nil or empty!!", cldDrvInfo.DriverName)
	}
	driverPath := driverLibPath + driverFile

	cblog.Info(cldDrvInfo.DriverName + ": driver path - " + driverPath)

	/*---------------
	        A plugin is only initialized once, and cannot be closed.
	        ref) https://golang.org/pkg/plugin/
	-----------------*/

	//var plug *plugin.Plugin
	plug, err := plugin.Open(driverPath)
	if err != nil {
		cblog.Errorf("plugin.Open: %v\n", err)
		return nil, err
	}
	//      fmt.Printf("plug: %#v\n\n", plug)

	//driver, err := plug.Lookup(cccInfo.DriverName)
	driver, err := plug.Lookup("CloudDriver")
	if err != nil {
		cblog.Errorf("plug.Lookup: %v\n", err)
		return nil, err
	}

	cloudDriver, ok := driver.(idrv.CloudDriver)
	if !ok {
		cblog.Error("Not CloudDriver interface!!")
		return nil, err
	}

	return cloudDriver, nil
}

func getStaticCloudDriver(cldDrvInfo dim.CloudDriverInfo) (idrv.CloudDriver, error) {
	cblog.Info("CloudDriverHandler: called getStaticCloudDriver() - " + cldDrvInfo.DriverName)

	var cloudDriver idrv.CloudDriver

	// select driver
	switch cldDrvInfo.ProviderName {
	case "AWS":
		cloudDriver = new(awsdrv.AwsDriver)
	case "AZURE":
		cloudDriver = new(azuredrv.AzureDriver)
	case "GCP":
		cloudDriver = new(gcpdrv.GCPDriver)
	case "ALIBABA":
		cloudDriver = new(alibabadrv.AlibabaDriver)
	case "OPENSTACK":
		cloudDriver = new(openstackdrv.OpenStackDriver)
	case "CLOUDIT":
		cloudDriver = new(clouditdrv.ClouditDriver)
	case "DOCKER":
		cloudDriver = new(dockerdrv.DockerDriver)
	//case "CLOUDTWIN":
	//	cloudDriver = new(cloudtwindrv.CloudTwinDriver)
	case "NCP": // NCP
		cloudDriver = new(ncpdrv.NcpDriver) // NCP

	default:
		errmsg := cldDrvInfo.ProviderName + " is not supported static Cloud Driver!!"
		return cloudDriver, fmt.Errorf(errmsg)
	}

	return cloudDriver, nil
}
