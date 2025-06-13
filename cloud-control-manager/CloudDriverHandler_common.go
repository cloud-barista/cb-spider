// Cloud Driver Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.12.

package clouddriverhandler

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	icdrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	im "github.com/cloud-barista/cb-spider/cloud-info-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"

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

// 1. get the driver info
// 2. get CloudDriver
func getCloudDriverByDriverName(driverName string) (idrv.CloudDriver, error) {
	cldDrvInfo, err := dim.GetCloudDriver(driverName)
	if err != nil {
		return nil, err
	}

	return getCloudDriver(*cldDrvInfo)
}

func GetProviderNameByDriverName(driverName string) (string, error) {
	cldDrvInfo, err := dim.GetCloudDriver(driverName)
	if err != nil {
		return "", err
	}

	return cldDrvInfo.ProviderName, nil
}

// CloudConnection for Region-Level Control (Except. DiskHandler)
func GetCloudConnection(cloudConnectName string) (icon.CloudConnection, error) {
	var conn icon.CloudConnection
	var err error

	for i := 0; i < 3; i++ {
		conn, err = commonGetCloudConnection(cloudConnectName, "")
		if err == nil {
			return conn, nil
		}

		if !strings.Contains(err.Error(), "dial") {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, err
}

// CloudConnection for Zone-Level Control (Ex. DiskHandler)
func GetZoneLevelCloudConnection(cloudConnectName string, targetZoneName string) (icon.CloudConnection, error) {
	var conn icon.CloudConnection
	var err error

	for i := 0; i < 3; i++ {
		conn, err = commonGetCloudConnection(cloudConnectName, targetZoneName)
		if err == nil {
			return conn, nil
		}

		if !strings.Contains(err.Error(), "dial") {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, err
}

func commonGetCloudConnection(cloudConnectName string, targetZoneName string) (icon.CloudConnection, error) {
	// Get cloud driver
	cldDriver, err := GetCloudDriver(cloudConnectName)
	if err != nil {
		return nil, err
	}

	// Get connection info using the new function
	connectionInfo, err := createConnectionInfo(cloudConnectName, targetZoneName)
	if err != nil {
		return nil, err
	}

	// Connect to the cloud using the connection info
	cldConnection, err := cldDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}

	return cldConnection, nil
}

// Create ConnectionInfo object
func createConnectionInfo(cloudConnectName string, targetZoneName string) (idrv.ConnectionInfo, error) {
	// Get connection configuration
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
	if err != nil {
		return idrv.ConnectionInfo{}, err
	}

	// Get decrypted credential information
	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return idrv.ConnectionInfo{}, err
	}

	// Get region information
	rgnInfo, err := rim.GetRegion(cccInfo.RegionName)
	if err != nil {
		return idrv.ConnectionInfo{}, err
	}

	// Extract region and zone names
	regionName, zoneName, err := getRegionNameByRegionInfo(rgnInfo)
	if err != nil {
		return idrv.ConnectionInfo{}, err
	}

	// Create connection info object
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:         getValue(crdInfo.KeyValueInfoList, "ClientId"),
			ClientSecret:     getValue(crdInfo.KeyValueInfoList, "ClientSecret"),
			StsToken:         getValue(crdInfo.KeyValueInfoList, "StsToken"),
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
		RegionInfo: idrv.RegionInfo{
			Region:     regionName,
			Zone:       zoneName,
			TargetZone: targetZoneName,
		},
	}

	return connectionInfo, nil
}

type CloudDriverAndConnectionInfo struct {
	CloudDriverInfo dim.CloudDriverInfo `json:"CloudDriverInfo"`
	ConnectionInfo  idrv.ConnectionInfo `json:"ConnectionInfo"`
}

// Critical data for Spiderlet.
// Requires a TLS environment.
func GetCloudDriverAndConnectionInfo(connectName string) (CloudDriverAndConnectionInfo, error) {
	// Get ConnectionConfig
	cccInfo, err := ccim.GetConnectionConfig(connectName)
	if err != nil {
		return CloudDriverAndConnectionInfo{}, err
	}

	// Get CloudDriverInfo
	cloudDriverInfo, err := dim.GetCloudDriver(cccInfo.DriverName)
	if err != nil {
		return CloudDriverAndConnectionInfo{}, err
	}

	// Create ConnectionInfo using the helper function
	connectionInfo, err := createConnectionInfo(connectName, "")
	if err != nil {
		return CloudDriverAndConnectionInfo{}, err
	}

	// Create and return ExportCloudInfo
	drvConnInfo := CloudDriverAndConnectionInfo{
		CloudDriverInfo: *cloudDriverInfo,
		ConnectionInfo:  connectionInfo,
	}

	return drvConnInfo, nil
}

// // for spiderlet
// CreateCloudConnection retrieves CloudDriverInfo and ConnectionInfo via HTTPS and constructs CloudConnection
func CreateCloudConnection(connectName string) (icon.CloudConnection, error) {
	// Set up curl command
	basePath := os.Getenv("CBSPIDER_ROOT")
	lionKeyPath := filepath.Join(basePath, "spiderlet", "lionkey")
	certFile := filepath.Join(lionKeyPath, "lionkey.crt")
	keyFile := filepath.Join(lionKeyPath, "lionkey.key")
	caCertFile := filepath.Join(lionKeyPath, "cert.pem")

	// Use curl to make the HTTPS request
	apiURL := fmt.Sprintf("https://localhost:10241/getcredentials/%s", connectName)
	cmd := exec.Command("curl", "--silent", "--cacert", caCertFile, "--cert", certFile, "--key", keyFile, apiURL)

	// Execute curl command and capture output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to call API using curl: %v", err)
	}

	// Parse the output JSON
	var apiResponse struct {
		CloudDriverInfo struct {
			DriverLibFileName string `json:"DriverLibFileName"`
			DriverName        string `json:"DriverName"`
			ProviderName      string `json:"ProviderName"`
		} `json:"CloudDriverInfo"`
		ConnectionInfo struct {
			CredentialInfo struct {
				APIVersion       string `json:"APIVersion"`
				ApiKey           string `json:"ApiKey"`
				AuthToken        string `json:"AuthToken"`
				ClientEmail      string `json:"ClientEmail"`
				ClientId         string `json:"ClientId"`
				ClientSecret     string `json:"ClientSecret"`
				StsToken         string `json:"StsToken"`
				ClusterId        string `json:"ClusterId"`
				ConnectionName   string `json:"ConnectionName"`
				DomainName       string `json:"DomainName"`
				Host             string `json:"Host"`
				IdentityEndpoint string `json:"IdentityEndpoint"`
				MockName         string `json:"MockName"`
				Password         string `json:"Password"`
				PrivateKey       string `json:"PrivateKey"`
				ProjectID        string `json:"ProjectID"`
				SubscriptionId   string `json:"SubscriptionId"`
				TenantId         string `json:"TenantId"`
				Username         string `json:"Username"`
			} `json:"CredentialInfo"`
			RegionInfo struct {
				Region     string `json:"Region"`
				Zone       string `json:"Zone"`
				TargetZone string `json:"TargetZone"`
			} `json:"RegionInfo"`
		} `json:"ConnectionInfo"`
	}

	// Unmarshal JSON response
	err = json.Unmarshal(output, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	// Construct CloudDriverInfo
	cloudDriverInfo := dim.CloudDriverInfo{
		DriverName:        apiResponse.CloudDriverInfo.DriverName,
		ProviderName:      apiResponse.CloudDriverInfo.ProviderName,
		DriverLibFileName: apiResponse.CloudDriverInfo.DriverLibFileName,
	}

	// Construct ConnectionInfo
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:         apiResponse.ConnectionInfo.CredentialInfo.ClientId,
			ClientSecret:     apiResponse.ConnectionInfo.CredentialInfo.ClientSecret,
			StsToken:         apiResponse.ConnectionInfo.CredentialInfo.StsToken,
			ConnectionName:   apiResponse.ConnectionInfo.CredentialInfo.ConnectionName,
			APIVersion:       apiResponse.ConnectionInfo.CredentialInfo.APIVersion,
			ApiKey:           apiResponse.ConnectionInfo.CredentialInfo.ApiKey,
			AuthToken:        apiResponse.ConnectionInfo.CredentialInfo.AuthToken,
			ClientEmail:      apiResponse.ConnectionInfo.CredentialInfo.ClientEmail,
			ClusterId:        apiResponse.ConnectionInfo.CredentialInfo.ClusterId,
			DomainName:       apiResponse.ConnectionInfo.CredentialInfo.DomainName,
			Host:             apiResponse.ConnectionInfo.CredentialInfo.Host,
			IdentityEndpoint: apiResponse.ConnectionInfo.CredentialInfo.IdentityEndpoint,
			MockName:         apiResponse.ConnectionInfo.CredentialInfo.MockName,
			Password:         apiResponse.ConnectionInfo.CredentialInfo.Password,
			PrivateKey:       apiResponse.ConnectionInfo.CredentialInfo.PrivateKey,
			ProjectID:        apiResponse.ConnectionInfo.CredentialInfo.ProjectID,
			SubscriptionId:   apiResponse.ConnectionInfo.CredentialInfo.SubscriptionId,
			TenantId:         apiResponse.ConnectionInfo.CredentialInfo.TenantId,
			Username:         apiResponse.ConnectionInfo.CredentialInfo.Username,
		},
		RegionInfo: idrv.RegionInfo{
			Region:     apiResponse.ConnectionInfo.RegionInfo.Region,
			Zone:       apiResponse.ConnectionInfo.RegionInfo.Zone,
			TargetZone: apiResponse.ConnectionInfo.RegionInfo.TargetZone,
		},
	}

	// Get the CloudDriver based on DriverName
	cloudDriver, err := getCloudDriver(cloudDriverInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get CloudDriver: %v", err)
	}

	// Use CloudDriver to create a CloudConnection
	cldConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudConnection: %v", err)
	}

	return cldConnection, nil
}

// 1. get credential info
// 2. get region info
// 3. get CloudConneciton
func GetCloudConnectionByDriverNameAndCredentialName(driverName string, credentialName string) (icon.CloudConnection, error) {
	cldDriver, err := getCloudDriverByDriverName(driverName)
	if err != nil {
		return nil, err
	}

	crdInfo, err := cim.GetCredentialDecrypt(credentialName)
	if err != nil {
		return nil, err
	}

	providerName, err := GetProviderNameByDriverName(driverName)
	if err != nil {
		return nil, err
	}

	connectionInfo := idrv.ConnectionInfo{ // @todo powerkim
		CredentialInfo: idrv.CredentialInfo{
			ClientId:         getValue(crdInfo.KeyValueInfoList, "ClientId"),
			ClientSecret:     getValue(crdInfo.KeyValueInfoList, "ClientSecret"),
			StsToken:         getValue(crdInfo.KeyValueInfoList, "StsToken"),
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
		},
	}

	// get Provider's Meta Info for default region
	cloudOSMetaInfo, err := im.GetCloudOSMetaInfo(providerName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if cloudOSMetaInfo.DefaultRegionToQuery != nil {
		if Length := len(cloudOSMetaInfo.DefaultRegionToQuery); Length == 1 {
			connectionInfo.RegionInfo.Region = cloudOSMetaInfo.DefaultRegionToQuery[0]
		} else if Length == 2 {
			connectionInfo.RegionInfo.Region = cloudOSMetaInfo.DefaultRegionToQuery[0]
			connectionInfo.RegionInfo.Zone = cloudOSMetaInfo.DefaultRegionToQuery[1]
		}
	}

	cldConnection, err := cldDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}

	return cldConnection, nil
}

func ListConnectionNameByProviderAndRegion(provider string, region string) ([]string, error) {
	connectionConfigInfoList, err := ccim.ListConnectionConfig()
	if err != nil {
		return nil, err
	}

	connectionNameList := []string{}
	for _, connectionConfigInfo := range connectionConfigInfoList {
		if connectionConfigInfo.ProviderName == provider {
			targetRegion, _, err := GetRegionNameByConnectionName(connectionConfigInfo.ConfigName)
			if err != nil {
				return nil, err
			}
			if targetRegion == region {
				connectionNameList = append(connectionNameList, connectionConfigInfo.ConfigName)
			}
		}
	}

	return connectionNameList, nil
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
	case "AWS", "AZURE", "ALIBABA", "GCP", "TENCENT", "IBM", "OPENSTACK", "NCP", "NCPVPC", "KTCLOUD", "NHNCLOUD", "KTCLOUDVPC":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
		zoneName = getValue(rgnInfo.KeyValueInfoList, "Zone")
	case "CLOUDTWIN", "MOCK":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	default:
		errmsg := rgnInfo.ProviderName + " is not a valid ProviderName!!"
		return "", "", fmt.Errorf(errmsg)
	}

	return regionName, zoneName, nil
}

func getValue(keyValueInfoList []icdrs.KeyValue, key string) string {
	for _, kv := range keyValueInfoList {
		if strings.EqualFold(kv.Key, key) { // ignore case
			return kv.Value
		}
	}
	return "Not set"
}
