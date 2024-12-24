// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.09.

package commonruntime

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"gopkg.in/yaml.v2"
)

// ================ RegionZone Handler
func ListRegionZone(connectionName string) ([]*cres.RegionZoneInfo, error) {
	cblog.Info("call ListRegionZone()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	infoList, err := handler.ListRegionZone()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if infoList == nil || len(infoList) <= 0 {
		infoList = []*cres.RegionZoneInfo{}
		return infoList, nil
	}

	// Set KeyValueList to an empty array if it is nil
	for _, region := range infoList {
		if region.KeyValueList == nil {
			region.KeyValueList = []cres.KeyValue{}
		}
	}

	// Update DisplayName and CSPDisplayName from metadata files
	cspName, err := ccm.GetProviderNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	csp := strings.ToLower(cspName)
	infoList, err = UpdateRegionZoneDisplayNames(csp, infoList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return infoList, nil
}

func UpdateRegionZoneDisplayNames(csp string, infoList []*cres.RegionZoneInfo) ([]*cres.RegionZoneInfo, error) {
	metaFile := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "cloud-driver-libs", "region", fmt.Sprintf("%s_region_meta.yaml", csp))
	data, err := ioutil.ReadFile(metaFile)
	if err != nil {
		// If the file does not exist, return without modification
		if strings.Contains(err.Error(), "no such file or directory") {
			cblog.Warnf("Metadata file for CSP '%s' not found. Skipping updates.", csp)
			return infoList, nil
		}
		return nil, err
	}

	var metadata struct {
		DisplayName    map[string]map[string]string `yaml:"DisplayName"`
		CSPDisplayName map[string]map[string]string `yaml:"CSPDisplayName"`
	}
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata file: %v", err)
	}

	for _, region := range infoList {
		// Update Region-level DisplayName and CSPDisplayName
		if regionDisplayName, exists := metadata.DisplayName[region.Name]; exists {
			if regionDisplay, ok := regionDisplayName[""]; ok {
				region.DisplayName = regionDisplay
			}
		}
		if regionCSPDisplayName, exists := metadata.CSPDisplayName[region.Name]; exists {
			if regionCSPDisplay, ok := regionCSPDisplayName[""]; ok {
				region.CSPDisplayName = regionCSPDisplay
			}
		}

		// Update Zone-level DisplayName and CSPDisplayName
		for i, zone := range region.ZoneList {
			if zoneDisplayName, exists := metadata.DisplayName[region.Name][zone.Name]; exists {
				region.ZoneList[i].DisplayName = zoneDisplayName
			}
			if zoneCSPDisplayName, exists := metadata.CSPDisplayName[region.Name][zone.Name]; exists {
				region.ZoneList[i].CSPDisplayName = zoneCSPDisplayName
			}
		}
	}

	return infoList, nil
}

func GetRegionZone(connectionName string, nameID string) (*cres.RegionZoneInfo, error) {
	cblog.Info("call GetRegionZone()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	info, err := handler.GetRegionZone(nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Set KeyValueList to an empty array if it is nil
	if info.KeyValueList == nil {
		info.KeyValueList = []cres.KeyValue{}
	}

	// Update DisplayName and CSPDisplayName from metadata files
	cspName, err := ccm.GetProviderNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	csp := strings.ToLower(cspName)
	updatedInfoList, err := UpdateRegionZoneDisplayNames(csp, []*cres.RegionZoneInfo{&info})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return updatedInfoList[0], nil
}

func ListOrgRegion(connectionName string) (string, error) {
	cblog.Info("call ListOrgRegion()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	infoList, err := handler.ListOrgRegion()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return infoList, nil
}

func ListOrgZone(connectionName string) (string, error) {
	cblog.Info("call GetOrgRegionZone()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	info, err := handler.ListOrgZone()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}

// ================ RegionZone Handler (Pre-Config Version)

func ListRegionZonePreConfig(driverName string, credentialName string) ([]*cres.RegionZoneInfo, error) {
	cblog.Info("call ListRegionZonePreConfig()")

	// check empty and trim user inputs
	driverName, err := EmptyCheckAndTrim("driverName", driverName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	credentialName, err = EmptyCheckAndTrim("credentialName", credentialName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnectionByDriverNameAndCredentialName(driverName, credentialName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	infoList, err := handler.ListRegionZone()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if infoList == nil || len(infoList) <= 0 {
		infoList = []*cres.RegionZoneInfo{}
	}

	// Set KeyValueList to an empty array if it is nil
	for _, region := range infoList {
		if region.KeyValueList == nil {
			region.KeyValueList = []cres.KeyValue{}
		}
	}

	// Update DisplayName and CSPDisplayName from metadata files
	cspName, err := ccm.GetProviderNameByDriverName(driverName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	csp := strings.ToLower(cspName)
	infoList, err = UpdateRegionZoneDisplayNames(csp, infoList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return infoList, nil
}

func GetRegionZonePreConfig(driverName string, credentialName string, nameID string) (*cres.RegionZoneInfo, error) {
	cblog.Info("call GetRegionZonePreConfig()")

	driverName, err := EmptyCheckAndTrim("driverName", driverName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	credentialName, err = EmptyCheckAndTrim("credentialName", credentialName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnectionByDriverNameAndCredentialName(driverName, credentialName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	info, err := handler.GetRegionZone(nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Set KeyValueList to an empty array if it is nil
	if info.KeyValueList == nil {
		info.KeyValueList = []cres.KeyValue{}
	}

	// Update DisplayName and CSPDisplayName from metadata files
	cspName, err := ccm.GetProviderNameByDriverName(driverName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	csp := strings.ToLower(cspName)
	updatedInfoList, err := UpdateRegionZoneDisplayNames(csp, []*cres.RegionZoneInfo{&info})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return updatedInfoList[0], nil
}

func ListOrgRegionPreConfig(driverName string, credentialName string) (string, error) {
	cblog.Info("call ListOrgRegionPreConfig()")

	driverName, err := EmptyCheckAndTrim("driverName", driverName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	credentialName, err = EmptyCheckAndTrim("credentialName", credentialName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	cldConn, err := ccm.GetCloudConnectionByDriverNameAndCredentialName(driverName, credentialName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	infoList, err := handler.ListOrgRegion()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return infoList, nil
}
